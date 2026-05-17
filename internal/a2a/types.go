// Package a2a provides types and broker logic for the Agent-to-Agent protocol.
package a2a

// AgentCard is the discovery document served at /.well-known/agent.json.
// It describes the agent's identity, capabilities, and available skills.
type AgentCard struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	URL          string            `json:"url"`
	Version      string            `json:"version,omitempty"`
	Skills       []Skill           `json:"skills"`
	Capabilities *AgentCapabilities `json:"capabilities,omitempty"`
	Auth         *AgentAuth        `json:"auth,omitempty"`
}

// Skill describes a discrete capability advertised in an AgentCard.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// AgentCapabilities declares optional protocol features the agent supports.
type AgentCapabilities struct {
	Streaming         bool `json:"streaming,omitempty"`
	PushNotifications bool `json:"pushNotifications,omitempty"`
	StateTransition   bool `json:"stateTransitionHistory,omitempty"`
}

// AgentAuth describes the authentication scheme required to call the agent.
type AgentAuth struct {
	Schemes []string `json:"schemes"`
}
