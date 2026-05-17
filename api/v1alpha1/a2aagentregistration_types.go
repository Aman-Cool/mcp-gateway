package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=a2aar
// +kubebuilder:printcolumn:name="SkillPrefix",type="string",JSONPath=".spec.skillPrefix",description="Prefix applied to federated skill IDs"
// +kubebuilder:printcolumn:name="Target",type="string",JSONPath=".spec.targetRef.name",description="Target HTTPRoute"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="Ready status"
// +kubebuilder:printcolumn:name="Skills",type="integer",JSONPath=".status.discoveredSkills",description="Number of discovered skills"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// A2AAgentRegistration registers an upstream A2A agent with the gateway.
// The gateway fetches the agent's card at /.well-known/agent.json, prefixes
// its skill IDs, and includes them in the federated card it serves.
type A2AAgentRegistration struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of A2AAgentRegistration.
	// +optional
	Spec A2AAgentRegistrationSpec `json:"spec,omitempty"`

	// status defines the observed state of A2AAgentRegistration.
	// +optional
	Status A2AAgentRegistrationStatus `json:"status,omitempty"`
}

// A2AAgentRegistrationSpec defines the desired state of A2AAgentRegistration.
type A2AAgentRegistrationSpec struct {
	// targetRef specifies the HTTPRoute pointing to the upstream A2A agent.
	// +required
	TargetRef TargetReference `json:"targetRef,omitzero"`

	// skillPrefix is prepended to each skill ID in the federated card to avoid
	// naming conflicts across agents.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="skillPrefix is immutable once set"
	SkillPrefix string `json:"skillPrefix,omitempty"`

	// agentCardURL overrides the default /.well-known/agent.json path when the
	// upstream agent serves its card at a non-standard URL.
	// +optional
	AgentCardURL string `json:"agentCardURL,omitempty"`

	// credentialRef references a Secret containing credentials for the upstream agent.
	// +optional
	CredentialRef *SecretReference `json:"credentialRef,omitempty"`
}

// A2AAgentRegistrationStatus defines the observed state of A2AAgentRegistration.
type A2AAgentRegistrationStatus struct {
	// conditions represent the latest available observations of A2AAgentRegistration state.
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// discoveredSkills is the number of skills discovered from the upstream agent card.
	// +optional
	DiscoveredSkills int32 `json:"discoveredSkills,omitempty"`
}

// +kubebuilder:object:root=true

// A2AAgentRegistrationList contains a list of A2AAgentRegistration.
type A2AAgentRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []A2AAgentRegistration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&A2AAgentRegistration{}, &A2AAgentRegistrationList{})
}
