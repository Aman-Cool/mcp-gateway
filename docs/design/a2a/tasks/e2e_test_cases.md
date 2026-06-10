# A2A E2E Test Cases

> Test cases follow the format defined in `tests/e2e/test_cases.md`.
> Tags: `Happy` (PR gate), `A2A` (A2A feature suite), `A2ASecurity` (auth/security paths).

---

### [Happy,A2A] Agent card discovery returns federated skills from registered agents

When an `A2AAgentRegistration` is created with a valid HTTPRoute pointing to an A2A test server,
and the test server's Agent Card lists two skills, the gateway's `GET /.well-known/agent.json`
endpoint should return a federated Agent Card containing those two skills prefixed with the
registration's `skillPrefix`. The gateway name and description in the returned card should reflect
the gateway, not any individual upstream agent.

---

### [Happy,A2A] message/send routes to the correct upstream agent and returns a gateway task ID

When a client with a valid `mcp-session-id` sends a `message/send` request with a skill that
matches a registered agent's prefix, the gateway should route the request to the correct upstream
A2A agent, return a response containing a gateway-generated task ID (not the upstream's task ID),
and store the task route mapping. The upstream agent should receive the request with the upstream
task ID.

---

### [Happy,A2A] tasks/get resolves gateway task ID to upstream agent and returns task status

When a client sends a `tasks/get` request with a gateway task ID previously returned by
`message/send`, the gateway should resolve the task ID to the correct upstream agent, rewrite the
ID to the upstream task ID, forward the request, and return the upstream result to the client
with the gateway task ID restored in the response.

---

### [Happy,A2A] tasks/cancel propagates to upstream and returns canceled state

When a client sends a `tasks/cancel` request with a valid gateway task ID, the gateway should
route the request to the correct upstream agent with the upstream task ID, and the client should
receive a response reflecting the canceled task state.

---

### [Happy,A2A] SSE streaming delivers task updates with consistent gateway task IDs

When a client sends a `message/send` request with `Accept: text/event-stream`, the gateway should
deliver SSE chunks in real time. All `data:` events should contain the gateway task ID (not the
upstream task ID) in the `id` field. The stream should complete when the upstream agent sends a
terminal state (`completed`, `failed`, or `canceled`).

---

### [A2A] Agent deregistration removes skills from federated card within one reconcile cycle

When an `A2AAgentRegistration` is deleted, the skills it contributed should no longer appear in
`GET /.well-known/agent.json` within one reconcile cycle. A `message/send` request using a skill
from the deregistered agent should return JSON-RPC error `-32602` (unknown skill) after the
reconcile completes.

---

### [A2A] Multiple agents federated with distinct prefixes

When two `A2AAgentRegistrations` are created with different `skillPrefix` values, the federated
Agent Card should contain skills from both agents, each correctly prefixed. A `message/send`
request using a skill from agent A should route to agent A; a request using a skill from agent B
should route to agent B. There should be no cross-routing.

---

### [A2ASecurity] message/send without a valid session returns 401

When a client sends a `message/send` request to `/a2a` without a `mcp-session-id` header, or with
an expired or invalid JWT, the gateway should return 401 without forwarding anything to the
upstream agent. The upstream agent should receive no request.

---

### [A2ASecurity] message/send with unknown skill returns JSON-RPC -32602

When a client with a valid session sends a `message/send` request with a skill prefix that does
not match any registered `A2AAgentRegistration`, the gateway should return a JSON-RPC error
response with code `-32602` and not forward the request to any upstream agent.

---

### [A2ASecurity] x-a2a-agent header injected by client is stripped

When a client sends a request to `/a2a` with a manually-set `x-a2a-agent` header, the gateway
should strip this header before processing. The routing decision should be based solely on the
skill prefix in the request body, not on the injected header.

---

### [A2A] MCP tools/list and tools/call are unaffected by A2A changes

When A2A support is fully deployed (agents registered, broker serving `/.well-known/agent.json`,
router handling `/a2a`), a client performing MCP `tools/list` should receive the same federated
tool list as before. A `tools/call` request should route correctly to the MCP backend and return
the expected result. No regressions in MCP behavior.
