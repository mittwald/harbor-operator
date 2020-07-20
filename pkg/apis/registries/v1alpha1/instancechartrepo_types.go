package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RepoState string

const (
	RepoStateReady RepoState = "Ready"
	RepoStateError RepoState = "Error"
)

// InstanceChartRepoSpec defines the desired state of InstanceChartRepo.
type InstanceChartRepoSpec struct {
	URL string `json:"url"`

	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	SecretRef *v1.LocalObjectReference `json:"secretRef,omitempty"`
}

// InstanceChartRepoStatus defines the observed state of InstanceChartRepo.
type InstanceChartRepoStatus struct {
	State RepoState `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstanceChartRepo is the Schema for the instancechartrepos API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=instancechartrepos,scope=Namespaced
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.url",description="URL"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="status"
type InstanceChartRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceChartRepoSpec   `json:"spec,omitempty"`
	Status InstanceChartRepoStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstanceChartRepoList contains a list of InstanceChartRepo.
type InstanceChartRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanceChartRepo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstanceChartRepo{}, &InstanceChartRepoList{})
}
