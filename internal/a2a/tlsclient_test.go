package a2a

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
)

// tlsCardServer starts a TLS card server and returns it with its cert as PEM,
// so tests can grant trust via the per-agent CA or the gateway bundle.
func tlsCardServer(t *testing.T, namespace, prefix string) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(gatewayCard(namespace, prefix)))
	}))
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	return srv, string(pemBytes)
}

func TestRefreshCard_TLSUntrustedKeepsStale(t *testing.T) {
	srv, _ := tlsCardServer(t, "mcp-test", "weather")
	defer srv.Close()

	b := newTestBroker()
	a := agentFor(srv.URL)
	b.register(a)
	b.store.Set("mcp-test", "weather", CardEntry{Raw: []byte("stale")})
	b.refreshCard(context.Background(), "mcp-test", "weather", a)

	e, _ := b.store.Get("mcp-test", "weather")
	if string(e.Raw) != "stale" {
		t.Fatalf("an untrusted TLS upstream must keep the stale card, got %q", e.Raw)
	}
}

func TestRefreshCard_TLSWithAgentCA(t *testing.T) {
	srv, caPEM := tlsCardServer(t, "mcp-test", "weather")
	defer srv.Close()

	b := newTestBroker()
	a := agentFor(srv.URL)
	a.CACert = caPEM
	b.register(a)
	b.refreshCard(context.Background(), "mcp-test", "weather", a)

	if _, ok := b.store.Get("mcp-test", "weather"); !ok {
		t.Fatal("card must be fetched over TLS when the per-agent CA is trusted")
	}
}

func TestRefreshCard_TLSWithGatewayBundle(t *testing.T) {
	srv, caPEM := tlsCardServer(t, "mcp-test", "weather")
	defer srv.Close()

	b := newTestBroker()
	b.mu.Lock()
	b.caCertPEM = caPEM
	b.mu.Unlock()
	a := agentFor(srv.URL)
	b.register(a)
	b.refreshCard(context.Background(), "mcp-test", "weather", a)

	if _, ok := b.store.Get("mcp-test", "weather"); !ok {
		t.Fatal("card must be fetched over TLS when the gateway CA bundle covers the upstream")
	}
}

func TestClientFor_CacheSharingAndInvalidPEM(t *testing.T) {
	srv, caPEM := tlsCardServer(t, "x", "y")
	defer srv.Close()

	b := newTestBroker()
	a := agentFor("https://agent.example")
	a.CACert = caPEM

	c1, err := b.clientFor("mcp-test/weather", a)
	if err != nil {
		t.Fatalf("client build failed: %v", err)
	}
	c2, _ := b.clientFor("mcp-test/weather", a)
	if c1 != c2 {
		t.Fatal("client must be cached while trust inputs are unchanged")
	}

	changed := agentFor("https://agent.example")
	changed.CACert = "not pem"
	if _, err := b.clientFor("mcp-test/weather", changed); err == nil {
		t.Fatal("an invalid agent CA PEM must fail client construction")
	}

	plain := agentFor("http://agent.example")
	c3, err := b.clientFor("mcp-test/other", plain)
	if err != nil {
		t.Fatalf("default client lookup failed: %v", err)
	}
	if c3 != b.client {
		t.Fatal("agents without custom trust must share the default client")
	}
}
