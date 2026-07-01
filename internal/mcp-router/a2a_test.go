package mcprouter

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	extprochttp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_proc/v3"
)

func TestIsA2APath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/a2a/weather", true},
		{"/a2a/weather/.well-known/agent-card.json", true},
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

func TestParseA2AMethod(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		want    string
		wantErr bool
	}{
		{"send", `{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"message":{}}}`, "message/send", false},
		{"stream", `{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{}}`, "message/stream", false},
		{"missing method", `{"jsonrpc":"2.0","id":1}`, "", false},
		{"invalid json", `{not json`, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseA2AMethod([]byte(c.body))
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("method = %q, want %q", got, c.want)
			}
		})
	}
}

func TestIsA2AStreamingMethod(t *testing.T) {
	cases := []struct {
		method string
		want   bool
	}{
		{a2aMethodStreamMessage, true},
		{a2aMethodResubscribe, true},
		{a2aMethodSendMessage, false},
		{"tasks/get", false},
		{"tasks/cancel", false},
	}
	for _, c := range cases {
		if got := isA2AStreamingMethod(c.method); got != c.want {
			t.Errorf("isA2AStreamingMethod(%q) = %v, want %v", c.method, got, c.want)
		}
	}
}

func TestA2AModeOverride(t *testing.T) {
	if got := a2aModeOverride(false).ResponseBodyMode; got != extprochttp.ProcessingMode_BUFFERED {
		t.Errorf("non-streaming ResponseBodyMode = %v, want BUFFERED", got)
	}
	if got := a2aModeOverride(true).ResponseBodyMode; got != extprochttp.ProcessingMode_STREAMED {
		t.Errorf("streaming ResponseBodyMode = %v, want STREAMED", got)
	}
}

func TestRewriteA2ABufferedTaskID(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	ctx := context.Background()

	t.Run("rewrites task id", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"result":{"id":"task-abc","kind":"task","status":{"state":"completed"}}}`
		out := rewriteA2ABufferedTaskID(ctx, logger, []byte(body))
		var envelope struct {
			Result struct {
				ID string `json:"id"`
			} `json:"result"`
		}
		if err := json.Unmarshal(out, &envelope); err != nil {
			t.Fatalf("output not json: %v", err)
		}
		if envelope.Result.ID != "gw-task-abc" {
			t.Errorf("result.id = %q, want %q", envelope.Result.ID, "gw-task-abc")
		}
	})

	t.Run("preserves sibling result fields", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"result":{"id":"task-abc","artifacts":[{"parts":[{"kind":"file","file":{"bytes":"aGVhdnk="}}]}]}}`
		out := rewriteA2ABufferedTaskID(ctx, logger, []byte(body))
		if !strings.Contains(string(out), `"aGVhdnk="`) {
			t.Errorf("artifact bytes not preserved: %s", out)
		}
	})

	t.Run("passes through message result without id", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"result":{"kind":"message","messageId":"m-1"}}`
		out := rewriteA2ABufferedTaskID(ctx, logger, []byte(body))
		if string(out) != body {
			t.Errorf("body changed: %s", out)
		}
	})

	t.Run("passes through non-json", func(t *testing.T) {
		body := `not json`
		out := rewriteA2ABufferedTaskID(ctx, logger, []byte(body))
		if string(out) != body {
			t.Errorf("body changed: %s", out)
		}
	})

	t.Run("passes through error response", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"error":{"code":-32001,"message":"task not found"}}`
		out := rewriteA2ABufferedTaskID(ctx, logger, []byte(body))
		if string(out) != body {
			t.Errorf("body changed: %s", out)
		}
	})
}
