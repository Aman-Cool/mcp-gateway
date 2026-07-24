package mcprouter

import (
	"encoding/json"
	"testing"
)

func TestIsA2APath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/a2a/mcp-test/weather", true},
		{"/a2a/mcp-test/weather/.well-known/agent-card.json", true},
		{"/mcp", false},
		{"/a2a", false},
		{"/", false},
	}
	for _, c := range cases {
		if got := isA2APath(c.path); got != c.want {
			t.Errorf("isA2APath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestResolveA2APath(t *testing.T) {
	cases := []struct {
		name          string
		path          string
		wantNamespace string
		wantPrefix    string
		wantOK        bool
	}{
		{"agent path", "/a2a/mcp-test/weather", "mcp-test", "weather", true},
		{"trailing card segments ignored", "/a2a/mcp-test/weather/.well-known/agent-card.json", "mcp-test", "weather", true},
		{"trailing slash", "/a2a/mcp-test/weather/", "mcp-test", "weather", true},
		{"query string ignored", "/a2a/mcp-test/weather?x=1", "mcp-test", "weather", true},
		{"missing prefix", "/a2a/mcp-test", "", "", false},
		{"empty", "/a2a/", "", "", false},
		{"double slash", "/a2a//weather", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ns, prefix, ok := resolveA2APath(c.path)
			if ok != c.wantOK || ns != c.wantNamespace || prefix != c.wantPrefix {
				t.Errorf("resolveA2APath(%q) = (%q, %q, %v), want (%q, %q, %v)", c.path, ns, prefix, ok, c.wantNamespace, c.wantPrefix, c.wantOK)
			}
		})
	}
}

func TestIsSupportedA2AMethod(t *testing.T) {
	cases := []struct {
		method string
		want   bool
	}{
		{a2aMethodSendMessage, true},
		{a2aMethodSendStreaming, true},
		{a2aMethodGetTask, true},
		{a2aMethodCancelTask, true},
		{a2aMethodSubscribeToTask, true},
		{"ListTasks", false},
		{"GetExtendedAgentCard", false},
		{"message/send", false}, // v0.3 name, not v1
		{"", false},
	}
	for _, c := range cases {
		if got := isSupportedA2AMethod(c.method); got != c.want {
			t.Errorf("isSupportedA2AMethod(%q) = %v, want %v", c.method, got, c.want)
		}
	}
}

func TestIsA2AStreamingMethod(t *testing.T) {
	cases := []struct {
		method string
		want   bool
	}{
		{a2aMethodSendStreaming, true},
		{a2aMethodSubscribeToTask, true},
		{a2aMethodSendMessage, false},
		{a2aMethodGetTask, false},
		{a2aMethodCancelTask, false},
	}
	for _, c := range cases {
		if got := isA2AStreamingMethod(c.method); got != c.want {
			t.Errorf("isA2AStreamingMethod(%q) = %v, want %v", c.method, got, c.want)
		}
	}
}

func TestParseA2ARequest(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		wantMethod  string
		wantHasPush bool
		wantErr     bool
	}{
		{"send message", `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"message":{}}}`, "SendMessage", false, false},
		{"get task", `{"jsonrpc":"2.0","id":"a","method":"GetTask","params":{"id":"t-1"}}`, "GetTask", false, false},
		{"embedded push config", `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"configuration":{"pushNotificationConfig":{"url":"https://x"}}}}`, "SendMessage", true, false},
		{"no push config", `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"configuration":{"blocking":true}}}`, "SendMessage", false, false},
		{"invalid json", `{not json`, "", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			method, _, hasPush, err := parseA2ARequest([]byte(c.body))
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if method != c.wantMethod {
				t.Errorf("method = %q, want %q", method, c.wantMethod)
			}
			if hasPush != c.wantHasPush {
				t.Errorf("hasPush = %v, want %v", hasPush, c.wantHasPush)
			}
		})
	}
}

func TestParseA2ARequestEchoesID(t *testing.T) {
	_, id, _, err := parseA2ARequest([]byte(`{"jsonrpc":"2.0","id":42,"method":"GetTask"}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(id) != "42" {
		t.Errorf("id = %s, want 42", id)
	}
}

func TestA2AErrorBody(t *testing.T) {
	t.Run("echoes id and codes error", func(t *testing.T) {
		body := a2aErrorBody(json.RawMessage("42"), a2aErrUnsupportedOp, "nope")
		var resp a2aErrorResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("output not json: %v", err)
		}
		if resp.JSONRPC != "2.0" || string(resp.ID) != "42" || resp.Error.Code != a2aErrUnsupportedOp || resp.Error.Message != "nope" {
			t.Errorf("unexpected error body: %s", body)
		}
	})

	t.Run("null id when unknown", func(t *testing.T) {
		body := a2aErrorBody(nil, a2aErrInvalidParams, "no agent")
		var resp a2aErrorResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("output not json: %v", err)
		}
		if string(resp.ID) != "null" {
			t.Errorf("id = %s, want null", resp.ID)
		}
	})
}
