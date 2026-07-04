package mcprouter

// a2a spike: derisks the per-method response ModeOverride (BUFFERED for
// non-streaming methods, STREAMED for sse methods) chosen at the
// response-headers phase. not production routing: agent resolution and
// task-id mapping are stubbed. see docs/design/a2a/a2a-design.md.

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	extprochttp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_proc/v3"
)

const a2aPathPrefix = "/a2a/"

const (
	a2aMethodSendMessage   = "message/send"
	a2aMethodStreamMessage = "message/stream"
	a2aMethodResubscribe   = "tasks/resubscribe"
)

// a2aSpikeTaskIDPrefix makes the buffered rewrite observable from the
// client: a task id with this prefix proves the mutation happened in envoy.
const a2aSpikeTaskIDPrefix = "gw-"

// a2aState is per-stream state for an in-flight a2a request, kept separate
// from mcpRequest so the two protocols never share parsing state.
type a2aState struct {
	method    string
	streaming bool
}

func isA2APath(path string) bool {
	return strings.HasPrefix(path, a2aPathPrefix)
}

func isA2AStreamingMethod(method string) bool {
	return method == a2aMethodStreamMessage || method == a2aMethodResubscribe
}

// parseA2AMethod reads only the json-rpc envelope; params are never
// unmarshalled (envelope-only discipline from the design).
func parseA2AMethod(body []byte) (string, error) {
	var envelope struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", err
	}
	return envelope.Method, nil
}

// a2aModeOverride returns the per-method processing mode: BUFFERED for
// non-streaming methods so the full response body can be rewritten in one
// pass, STREAMED for sse methods so chunks flow through as they arrive.
func a2aModeOverride(streaming bool) *extprochttp.ProcessingMode {
	bodyMode := extprochttp.ProcessingMode_BUFFERED
	if streaming {
		bodyMode = extprochttp.ProcessingMode_STREAMED
	}
	return &extprochttp.ProcessingMode{
		RequestHeaderMode:   extprochttp.ProcessingMode_SEND,
		ResponseHeaderMode:  extprochttp.ProcessingMode_SEND,
		ResponseBodyMode:    bodyMode,
		RequestTrailerMode:  extprochttp.ProcessingMode_SKIP,
		ResponseTrailerMode: extprochttp.ProcessingMode_SKIP,
	}
}

// rewriteA2ABufferedTaskID rewrites result.id in a buffered non-streaming
// a2a response, proving a BUFFERED body selected mid-request arrives whole
// and is mutable. only the id key is decoded; sibling result fields stay
// raw. any parse failure passes the body through untouched.
func rewriteA2ABufferedTaskID(ctx context.Context, logger *slog.Logger, body []byte) []byte {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		logger.DebugContext(ctx, "a2a spike: response not json, passing through", "error", err)
		return body
	}
	resultRaw, ok := envelope["result"]
	if !ok {
		return body
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		return body
	}
	idRaw, ok := result["id"]
	if !ok {
		// message/send may return a plain message (no task): pass through
		return body
	}
	var id string
	if err := json.Unmarshal(idRaw, &id); err != nil || id == "" {
		return body
	}
	newID, err := json.Marshal(a2aSpikeTaskIDPrefix + id)
	if err != nil {
		return body
	}
	result["id"] = newID
	newResult, err := json.Marshal(result)
	if err != nil {
		return body
	}
	envelope["result"] = newResult
	newBody, err := json.Marshal(envelope)
	if err != nil {
		return body
	}
	logger.DebugContext(ctx, "a2a spike: rewrote buffered task id", "task", id)
	return newBody
}
