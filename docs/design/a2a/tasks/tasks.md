# A2A Protocol Support — Implementation Plan

## Existing Code Analysis

The following primitives exist in the codebase and are reused directly by the A2A implementation:

| Primitive | Location | Reused for |
|---|---|---|
| ext_proc Process() loop | `internal/mcp-router/server.go` | A2A traffic detection and routing |
| ResponseBuilder | `internal/mcp-router/response_builder.go` | Building all ext_proc responses |
| HeadersBuilder | `internal/mcp-router/headers.go` | Setting routing headers |
| sseRewriter | `internal/mcp-router/elicitation.go` | Template for a2aSSEPassthrough |
| idmap.Map | `internal/idmap/map.go` | Template for TaskStore (same in-memory/Redis duality) |
| session.Cache | `internal/session/cache.go` | Extended with TaskStore methods |
| JWTManager.Validate() | `internal/session/jwt.go` | Session validation for A2A requests |
| JWTManager.GetExpiresIn() | `internal/session/jwt.go` | TTL source for task store Redis keys |
| config.Observer | `internal/config/types.go` | A2A broker registers as observer |
| MCPServersConfig.Notify() | `internal/config/types.go` | Triggers A2A broker config updates |
| SecretReaderWriter | `internal/config/config_writer.go` | Extended with UpsertA2AAgent/RemoveA2AAgent |
| MCPReconciler | `internal/controller/mcpserverregistration_controller.go` | Template for A2AReconciler |
| HTTPRouteWrapper | `internal/controller/httproute_wrapper.go` | Used directly in A2AReconciler |
| buildGatewayHTTPRoute() | `internal/controller/broker_router.go` | Modified to add /a2a and /.well-known/agent.json rules |
| ModeOverride (SSE) | `internal/mcp-router/response_handlers.go` | A2A streaming passthrough |

---

## Phase 1: Foundation (Weeks 1–4)

### Task 1: Design Document + Gap Analysis

**Files:**
- `docs/design/a2a/a2a-design.md` (this document's companion)
- `docs/design/a2a/tasks/tasks.md` (this file)
- `docs/design/a2a/tasks/e2e_test_cases.md`

**Acceptance criteria:**
- [ ] Design doc covers all sections per `docs/design/CLAUDE.md` structure
- [ ] Mermaid diagrams for agent card discovery, message/send routing, task lifecycle
- [ ] Open design questions surfaced as explicit tradeoff analyses
- [ ] `make spell` passes
- [ ] Mentors approve design before Task 2 begins

**Verification:**
```bash
make spell
```

---

### Task 2: A2A Test Server

**Files:**
- `tests/servers/a2a-server/main.go`
- `tests/servers/a2a-server/Dockerfile`
- `config/test-servers/a2a-server-deployment.yaml`
- `config/test-servers/a2a-server-service.yaml`
- `config/test-servers/a2a-server-httproute.yaml`
- `config/test-servers/kustomization.yaml` (updated)

**Acceptance criteria:**
- [ ] `GET /.well-known/agent.json` returns a valid AgentCard configurable via `AGENT_NAME`, `SKILLS`, `SKILL_PREFIX` env vars
- [ ] `POST /a2a` dispatches `message/send` (returns completed task), `tasks/get`, `tasks/cancel`
- [ ] SSE streaming for `message/send` when `Accept: text/event-stream`: three `working` events then `completed`
- [ ] Kubernetes manifests follow `config/test-servers/server1-deployment.yaml` pattern
- [ ] Server added to `config/test-servers/kustomization.yaml`

**Verification:**
```bash
curl http://a2a-server.mcp-test.svc.cluster.local:8080/.well-known/agent.json
curl -X POST http://a2a-server.mcp-test.svc.cluster.local:8080/a2a \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"message":{"role":"user","parts":[{"type":"text","text":"hello"}]}}}'
```

---

### Task 3: A2AAgentRegistration CRD Finalization

*Depends on: Task 1 (design doc approved, API group question answered)*

**Files:**
- `api/v1alpha1/a2aagentregistration_types.go`
- `api/v1alpha1/zz_generated.deepcopy.go` (regenerated)
- `config/crd/mcp.kuadrant.io_a2aagentregistrations.yaml` (regenerated)
- `charts/mcp-gateway/crds/mcp.kuadrant.io_a2aagentregistrations.yaml` (regenerated)
- `docs/reference/a2aagentregistration.md`

**Acceptance criteria:**
- [ ] `skillPrefix` immutability CEL rule passes `make lint`
- [ ] `agentCardURL` URL format validation present
- [ ] `targetRef` uses `omitzero` (kubeapilinter requirement)
- [ ] `make generate-all` produces no diff after this PR
- [ ] `kubectl apply -f config/crd/...a2aagentregistrations.yaml` succeeds against Kind cluster
- [ ] kubeapilinter passes in CI

**Verification:**
```bash
make generate-all
git diff --exit-code
kubectl apply -f config/crd/mcp.kuadrant.io_a2aagentregistrations.yaml
make lint
```

---

### Task 4: A2AReconciler Scaffold

**Files:**
- `internal/controller/a2aagentregistration_controller.go` (new — scaffold only)
- `cmd/main.go` (register A2AReconciler)

**Acceptance criteria:**
- [ ] `A2AReconciler` struct has `Client`, `Scheme`, `DirectAPIReader`, `ConfigReaderWriter`, `MCPExtFinderValidator` fields
- [ ] `Reconcile()` returns `ctrl.Result{}` — skeleton only
- [ ] `SetupWithManager()` watches `A2AAgentRegistration`, `HTTPRoute`, `Secret` with same predicates as `MCPReconciler`
- [ ] Uses distinct finalizer `"mcp.kuadrant.io/a2a-finalizer"`
- [ ] `make build` passes
- [ ] Controller starts without errors against Kind cluster

**Verification:**
```bash
make build
make deploy
kubectl logs -n mcp-system deployment/mcp-gateway-controller
```

---

### Task 5: A2AReconciler Reconcile Logic + Tests

*Depends on: Task 4*

**Files:**
- `internal/controller/a2aagentregistration_controller.go` (fill in reconcile logic)
- `internal/controller/a2aagentregistration_controller_test.go` (new)
- `internal/controller/a2aagentregistration_controller_integration_test.go` (new)

**Acceptance criteria:**
- [ ] Finalizer added on create, removed only after config is cleaned up
- [ ] `getTargetHTTPRoute()` resolves HTTPRoute using `WrapHTTPRoute()` + `Validate()`
- [ ] `buildA2AAgentConfig()` handles `IsHostnameBackend()` and `IsServiceBackend()` using existing helpers
- [ ] `UpsertA2AAgent()` called for each valid MCPGatewayExtension namespace
- [ ] Status conditions: `Ready=True` when agent card is discoverable, `AgentCardDiscovered=True` after successful HTTP GET to agent card URL
- [ ] `DiscoveredSkills int32` in status reflects skill count from last fetched Agent Card
- [ ] Controller integration tests: new registration → Ready=True; missing HTTPRoute → Ready=False; deletion removes config

**Verification:**
```bash
make test-controller-integration
```

---

## Phase 2: Core Components (Weeks 5–8, buffer at Week 6)

### Task 6: Config Plumbing + Hot-Reload

**Files:**
- `internal/config/types.go`
- `internal/config/a2a_types.go` (new)
- `internal/config/config_writer.go`

**Acceptance criteria:**
- [ ] `A2AAgents []*A2AAgent` added to `MCPServersConfig` with RWMutex protection
- [ ] `SetA2AAgents()`, `ListA2AAgents()` follow `SetServers()`/`ListServers()` pattern exactly
- [ ] `UpsertA2AAgent()` and `RemoveA2AAgent()` in `SecretReaderWriter` with retry-on-conflict
- [ ] `BrokerConfig` YAML schema gains `a2aAgents` key
- [ ] `Notify()` passes A2A agents to observers alongside MCP servers
- [ ] `go test -race ./internal/config/...` passes

**Verification:**
```bash
go test -race ./internal/config/...
make test-unit
```

---

### Task 7: A2A Broker — Observer Wiring

*Depends on: Task 6*

**Files:**
- `internal/a2a/broker.go` (finalize PoC, wire Observer)
- `internal/a2a/broker_test.go` (extend)

**Acceptance criteria:**
- [ ] `a2a.Broker` implements `config.Observer`: `OnConfigChange()` calls `SetAgents(cfg.ListA2AAgents())`
- [ ] `FederatedCard()` has OTel span `"a2a.FederatedCard"` with `agent.count` attribute, following `HandleToolCall()` pattern
- [ ] Unit tests: `OnConfigChange` triggers `SetAgents`; `FederatedCard` with unreachable upstream skips gracefully; `GetAgentInfo` longest-prefix matching
- [ ] `go test -race ./internal/a2a/...` passes

**Verification:**
```bash
go test -race ./internal/a2a/...
make test-unit
```

---

### Task 8: A2A Broker — Binary Wiring

*Depends on: Task 7*

**Files:**
- `cmd/mcp-broker-router/main.go`
- `cmd/mcp-broker-router/broker.go`
- `cmd/mcp-broker-router/router.go`

**Acceptance criteria:**
- [ ] `a2aBroker` initialized in `main.go` and registered as observer: `cfg.RegisterObserver(a2aBroker)`
- [ ] `/.well-known/agent.json` registered in `setUpHTTPServer()` after `/.well-known/oauth-protected-resource`
- [ ] `A2ABroker a2a.Broker` field added to `ExtProcServer` struct in `createRouter()`
- [ ] `make build` passes

**Verification:**
```bash
make build
make deploy
curl http://mcp.127-0-0-1.sslip.io:8001/.well-known/agent.json
# expect: {"skills":[]} (no agents registered yet)
```

---

### Task 9: Router — A2A Traffic Detection

*Depends on: Task 8*

**Files:**
- `internal/mcp-router/server.go`
- `internal/mcp-router/headers.go`
- `internal/headers/headers.go`
- `internal/controller/broker_router.go`

**Acceptance criteria:**
- [ ] `isA2A` bool set in `Process()` at `RequestHeaders` phase via `strings.HasPrefix(requestPath, "/a2a")`
- [ ] A2A header constants defined in `internal/headers/headers.go`: `A2AAgentHeader`, `A2ATaskIDHeader`, `A2AMethodHeader`
- [ ] `WithA2AAgent()`, `WithA2ATaskID()`, `WithA2AMethod()` added to `HeadersBuilder`
- [ ] `x-a2a-agent` and `x-a2a-task-id` added to `internalOnlyHeaders`
- [ ] `buildGatewayHTTPRoute()` gains `/a2a` rule with `stripRouterHeaders` and `/.well-known/agent.json` rule
- [ ] Stub `RouteA2ARequest()` returning empty pass-through
- [ ] Unit tests: mock ext_proc stream with `/a2a` path → `isA2A=true`; `/mcp` path → `isA2A=false`
- [ ] `make test-unit` passes

**Verification:**
```bash
make test-unit
make lint
```

---

### Task 10: Router — A2A Request Routing

*Depends on: Task 9, open question Q3 (path discrimination) and Q4 (session handling) resolved*

**Files:**
- `internal/mcp-router/request_handlers.go`
- `internal/mcp-router/request_handlers_test.go`

**Acceptance criteria:**
- [ ] `A2ARequest` struct: `ID any`, `JSONRPC string`, `Method string`, `Params map[string]any`
- [ ] `parseA2ARequest(body []byte) (*A2ARequest, error)`
- [ ] `RouteA2ARequest()`: validates session, switches on `message/send`/`tasks/get`/`tasks/cancel`
- [ ] `HandleA2ATaskSend()`: extracts skill → `A2ABroker.GetAgentInfo()` → sets `:authority`, `:path`, `x-a2a-agent` → `isStreamingMethod()` sets `ModeOverride`
- [ ] Unknown method → JSON-RPC `-32601`; missing session → 401; unknown skill → JSON-RPC `-32602`
- [ ] MCP path (`/mcp` traffic) completely unaffected — regression tests pass
- [ ] Unit tests cover all branches above

**Verification:**
```bash
make test-unit
# deploy and test end-to-end:
curl -X POST http://mcp.127-0-0-1.sslip.io:8001/a2a \
  -H "mcp-session-id: <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"skill":"weather_forecast","message":{...}}}'
```

---

## Phase 3: Integration & Hardening (Weeks 9–12)

### Task 11: Task ID Mapping

*Depends on: Task 10*

**Files:**
- `internal/session/cache.go`
- `internal/session/cache_test.go`

**Note:** The `a2a-task-routing-infra` branch has an existing partial implementation with a
simpler `StoreTaskRoute(ctx, taskID, serverName string)` signature. That branch must be rebased
onto current main and updated to use the full `TaskRoute` struct and gateway-owned task IDs
defined here before this task merges.

**Acceptance criteria:**
- [ ] `taskRoutes sync.Map` field added to `Cache` (separate from `inmemory`)
- [ ] `StoreTaskRoute(ctx, gatewayTaskID string, route TaskRoute) error` implemented for in-memory and Redis
- [ ] `ResolveTaskRoute(ctx, gatewayTaskID string) (TaskRoute, bool, error)` implemented for in-memory and Redis
- [ ] `DeleteTaskRoute(ctx, gatewayTaskID string) error` implemented for in-memory and Redis
- [ ] `SessionCache` interface in `internal/mcp-router/server.go` updated with the above signatures
- [ ] Redis key prefix `a2atask:`, TTL from `JWTManager.GetExpiresIn()` matching session TTL
- [ ] `HandleA2ATaskSend()` updated: generate gateway task ID, call `StoreTaskRoute()`, rewrite task ID in response body
- [ ] `HandleA2ATaskGet()` and `HandleA2ATaskCancel()` call `ResolveTaskRoute()` to find upstream agent and rewrite ID
- [ ] Concurrency test: 100 goroutines reading and writing task routes with `-race`
- [ ] `go test -race ./internal/session/...` passes

**Verification:**
```bash
go test -race ./internal/session/...
make test-unit
```

---

### Task 12: SSE Streaming Passthrough

*Depends on: Task 11*

**Files:**
- `internal/mcp-router/elicitation.go` (add `a2aSSEPassthrough`)
- `internal/mcp-router/response_handlers.go`
- `internal/mcp-router/server.go`

**Acceptance criteria:**
- [ ] `a2aSSEPassthrough` struct with `Process(ctx, chunk []byte) []byte` and `Flush(ctx) []byte`
- [ ] `Process()` rewrites upstream task IDs to gateway task IDs in `data:` lines (JSON field `"id"`)
- [ ] `HandleResponseHeaders()` sets `ModeOverride` with `ResponseBodyMode=STREAMED` when `isA2A && isStreamingA2AMethod()`
- [ ] `Process()` loop handles `a2aPassthrough` in `ResponseBody` phase identical to `rewriter`
- [ ] Unit tests: SSE chunks pass through; upstream task IDs replaced with gateway task IDs; non-SSE responses unaffected

**Verification:**
```bash
make test-unit
curl -X POST http://mcp.127-0-0-1.sslip.io:8001/a2a \
  -H "Accept: text/event-stream" \
  -H "mcp-session-id: <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"message/send","params":{...}}'
# expect: SSE stream with gateway task IDs in all events
```

---

### Task 13: E2E Tests — Discovery + Task Routing

*Depends on: Task 12*

**Files:**
- `tests/e2e/a2a_discovery_test.go`
- `tests/e2e/a2a_task_test.go`
- `tests/e2e/test_cases.md` (updated)

**Acceptance criteria:**
- [ ] Agent card discovery: `GET /.well-known/agent.json` returns federated card with prefixed skills from the A2A test server
- [ ] Task send: `message/send` with known skill routes to correct upstream, returns gateway task ID
- [ ] Task get: `tasks/get` with gateway task ID returns upstream result
- [ ] Task cancel: `tasks/cancel` propagates to upstream, returns canceled state
- [ ] Agent deregistration: deleting `A2AAgentRegistration` removes skills from federated card within one reconcile cycle

**Verification:**
```bash
ginkgo -v --label-filter="A2A" ./tests/e2e/...
```

---

### Task 14: E2E Tests — Streaming + Auth + Error + Regression

*Depends on: Task 13*

**Files:**
- `tests/e2e/a2a_discovery_test.go` (extend)
- `tests/e2e/a2a_task_test.go` (extend)

**Acceptance criteria:**
- [ ] Streaming: `message/send` with `Accept: text/event-stream` delivers SSE chunks with gateway task IDs
- [ ] Auth: request without valid `mcp-session-id` returns 401 before reaching upstream
- [ ] Unknown skill: `message/send` with unregistered skill prefix returns JSON-RPC `-32602`
- [ ] MCP regression: `tools/list` and `tools/call` work correctly after all A2A changes
- [ ] All E2E tests pass: `ginkgo -v ./tests/e2e/... -- --gateway-host=mcp.127-0-0-1.sslip.io:8001`

**Verification:**
```bash
ginkgo -v ./tests/e2e/... -- --gateway-host=mcp.127-0-0-1.sslip.io:8001
```

---

### Task 15: Documentation + Final Polish

**Files:**
- `docs/guides/a2a-agent.md`
- `docs/guides/README.md` (updated)

**Acceptance criteria:**
- [ ] Guide follows `docs/CLAUDE.md` conventions: goal-oriented, numbered steps, verification commands, no internal references
- [ ] Covers: prerequisites, Step 1 (HTTPRoute), Step 2 (A2AAgentRegistration), Step 3 (verify agent card), Step 4 (send a task), credentialRef usage
- [ ] Links to authentication guide for AuthPolicy on the `/a2a` path
- [ ] `make spell` passes
- [ ] Guide reviewed and approved by mentor

**Verification:**
```bash
make spell
```
