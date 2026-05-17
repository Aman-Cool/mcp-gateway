// Package config provides configuration types
package config

// A2AAgent represents an upstream A2A agent to be federated by the gateway.
type A2AAgent struct {
	Name        string      `json:"name"                  yaml:"name"`
	CardURL     string      `json:"cardURL"               yaml:"cardURL"`
	Hostname    string      `json:"hostname,omitempty"    yaml:"hostname,omitempty"`
	SkillPrefix string      `json:"skillPrefix,omitempty" yaml:"skillPrefix,omitempty"`
	Auth        *AuthConfig `json:"auth,omitempty"        yaml:"auth,omitempty"`
	Credential  string      `json:"credential,omitempty"  yaml:"credential,omitempty"`
	Enabled     bool        `json:"enabled"               yaml:"enabled"`
}
