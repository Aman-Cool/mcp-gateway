package a2a

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/Kuadrant/mcp-gateway/internal/config"
)

// agentClient caches a per-agent HTTP client together with the trust inputs it was built
// from, so a client is rebuilt only when the gateway bundle or the agent's CA changes.
type agentClient struct {
	client     *http.Client
	gatewayPEM string
	agentPEM   string
}

// clientFor returns the HTTP client for an agent's card fetches. Agents without any custom
// trust share the broker's default client; agents needing the gateway CA bundle and/or a
// per-agent CA get a dedicated client, cached until either trust input changes.
func (b *Broker) clientFor(key string, agent *config.A2AAgent) (*http.Client, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	gw := b.caCertPEM
	if gw == "" && agent.CACert == "" {
		return b.client, nil
	}
	if cc, ok := b.clients[key]; ok && cc.gatewayPEM == gw && cc.agentPEM == agent.CACert {
		return cc.client, nil
	}
	c, err := newCardTLSClient(gw, agent.CACert)
	if err != nil {
		return nil, err
	}
	b.clients[key] = &agentClient{client: c, gatewayPEM: gw, agentPEM: agent.CACert}
	return c, nil
}

// newCardTLSClient builds a card-fetch client whose trust pool is system roots plus the
// gateway-level CA bundle plus the per-agent CA — the same additive trust model the MCP
// broker uses for upstream connections.
func newCardTLSClient(gatewayPEM, agentPEM string) (*http.Client, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}
	if gatewayPEM != "" {
		if !rootCAs.AppendCertsFromPEM([]byte(gatewayPEM)) {
			return nil, fmt.Errorf("failed to parse gateway CA certificate bundle PEM")
		}
	}
	if agentPEM != "" {
		if !rootCAs.AppendCertsFromPEM([]byte(agentPEM)) {
			return nil, fmt.Errorf("failed to parse agent CA certificate PEM")
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    rootCAs,
	}
	return &http.Client{Timeout: cardFetchTimeout, Transport: transport}, nil
}
