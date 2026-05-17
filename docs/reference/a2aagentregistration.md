# The A2AAgentRegistration Custom Resource Definition (CRD)

- [A2AAgentRegistration](#a2aagentregistration)
- [A2AAgentRegistrationSpec](#a2aagentregistrationspec)
- [TargetReference](#targetreference)
- [SecretReference](#secretreference)
- [A2AAgentRegistrationStatus](#a2aagentregistrationstatus)

## A2AAgentRegistration

| **Field** | **Type** | **Required** | **Description** |
|-----------|----------|:------------:|-----------------|
| `spec` | [A2AAgentRegistrationSpec](#a2aagentregistrationspec) | Yes | The specification for A2AAgentRegistration custom resource |
| `status` | [A2AAgentRegistrationStatus](#a2aagentregistrationstatus) | No | The status for the custom resource |

## A2AAgentRegistrationSpec

| **Field** | **Type** | **Required** | **Description** |
|-----------|----------|:------------:|-----------------|
| `targetRef` | [TargetReference](#targetreference) | Yes | An HTTPRoute that points to the upstream A2A agent. The controller discovers the backend service from this HTTPRoute and configures the broker to federate its skills |
| `skillPrefix` | String | No | Prefix prepended to each skill ID in the federated card. Avoids naming conflicts when aggregating skills from multiple agents (e.g. `weather_forecast` and `search_forecast`). Immutable once set |
| `agentCardURL` | String | No | Overrides the default `/.well-known/agent.json` path when the upstream agent serves its card at a non-standard URL |
| `credentialRef` | [SecretReference](#secretreference) | No | Reference to a Secret containing authentication credentials for the upstream agent. The secret must have the label `mcp.kuadrant.io/secret=true` |

## TargetReference

| **Field** | **Type** | **Required** | **Description** |
|-----------|----------|:------------:|-----------------|
| `group` | String | No | Group of the target resource. Default: `gateway.networking.k8s.io` |
| `kind` | String | No | Kind of the target resource. Default: `HTTPRoute` |
| `name` | String | Yes | Name of the target HTTPRoute |
| `namespace` | String | No | Namespace of the target resource. Defaults to same namespace |

## SecretReference

| **Field** | **Type** | **Required** | **Description** |
|-----------|----------|:------------:|-----------------|
| `name` | String | Yes | Name of the Secret resource |
| `key` | String | No | Key within the Secret that contains the credential value. Default: `token` |

## A2AAgentRegistrationStatus

| **Field** | **Type** | **Description** |
|-----------|----------|-----------------|
| `conditions` | [][Kubernetes meta/v1.Condition](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition) | List of conditions that define the status of the resource |
| `discoveredSkills` | Integer | Number of skills discovered from the upstream agent card |
