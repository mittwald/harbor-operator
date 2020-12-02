package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RepoState string

const (
	RepoStateReady RepoState = "Ready"
	RepoStateError RepoState = "Error"
)

// InstanceChartRepositorySpec defines the desired state of an InstanceChartRepository.
type InstanceChartRepositorySpec struct {
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Optional
	SecretRef *v1.LocalObjectReference `json:"secretRef,omitempty"`
}

// InstanceChartRepositoryStatus defines the observed state of an InstanceChartRepository.
type InstanceChartRepositoryStatus struct {
	State RepoState `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstanceChartRepository is the Schema for the instancechartrepositories API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=instancechartrepositories,scope=Namespaced
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.url",description="URL"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="status"
type InstanceChartRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceChartRepositorySpec   `json:"spec,omitempty"`
	Status InstanceChartRepositoryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstanceChartRepositoryList contains a list of InstanceChartRepositories.
type InstanceChartRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanceChartRepository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstanceChartRepository{}, &InstanceChartRepositoryList{})
}
