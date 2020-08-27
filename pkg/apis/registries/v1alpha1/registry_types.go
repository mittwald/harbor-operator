package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	RegistryStatusPhaseName string
	RegistryType            string
)

const (
	RegistryStatusPhaseUnknown     RepositoryStatusPhaseName = ""
	RegistryStatusPhaseCreating    RepositoryStatusPhaseName = "Creating"
	RegistryStatusPhaseReady       RepositoryStatusPhaseName = "Ready"
	RegistryStatusPhaseTerminating RepositoryStatusPhaseName = "Terminating"
)

const (
	RegistryTypeHarbor           RegistryType = "harbor"
	RegistryTypeDockerHub        RegistryType = "docker-hub"
	RegistryTypeDockerRegistry   RegistryType = "docker-registry"
	RegistryTypeHuaweiSWR        RegistryType = "huawei-SWR"
	RegistryTypeGoogleGCR        RegistryType = "google-gcr"
	RegistryTypeAwsECR           RegistryType = "aws-ecr"
	RegistryTypeAzureECR         RegistryType = "azure-acr"
	RegistryTypeAliACR           RegistryType = "ali-acr"
	RegistryTypeJfrogArtifactory RegistryType = "jfrog-artifactory"
	RegistryTypeQuayIo           RegistryType = "quay-io"
	RegistryTypeGitlab           RegistryType = "gitlab"
	RegistryTypeHelmHub          RegistryType = "helm-hub"
)

// RegistrySpec defines the desired state of a Registry.
type RegistrySpec struct {
	Name string `json:"name"`

	// +optional
	Description string `json:"description,omitempty"`

	Type RegistryType `json:"type"`

	// Target URL of the registry
	URL string `json:"url"`

	// +optional
	Credential *RegistryCredential `json:"credential,omitempty"`

	// Whether or not the TLS certificate will be verified when Harbor tries to access the registry
	// +optional
	Insecure bool `json:"insecure,omitempty"`

	// ParentInstance is a LocalObjectReference to the
	// name of the harbor instance the registry is created for
	ParentInstance corev1.LocalObjectReference `json:"parentInstance"`
}

type RegistryCredential struct {
	// Secret reference to the credentials
	SecretRef corev1.LocalObjectReference `json:"secretRef"`

	// Key for the "AccessKey" field of the secret referenced in SecretRef
	SecretKeyAccessKey string `json:"secretKeyAccessKey"`

	// Key for the "AccessSecret" field of the secret referenced in SecretRef
	SecretKeyAccessSecret string `json:"secretKeyAccessSecret"`

	// Credential type, such as 'basic', 'oauth'.
	Type string `json:"type"`
}

// RegistryStatus defines the observed state of Registry.
type RegistryStatus struct {
	Name    string                    `json:"name"`
	Phase   RepositoryStatusPhaseName `json:"phase"`
	Message string                    `json:"message"`

	// Time of last observed transition into this state
	// +optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`

	// The registry ID is written back from the held registry ID.
	ID int64 `json:"id,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Registry is the Schema for the registries API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=registries,scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="phase"
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id",description="harbor registry id"
type Registry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RegistrySpec `json:"spec,omitempty"`

	Status RegistryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RegistryList contains a list of Registry
type RegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Registry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Registry{}, &RegistryList{})
}
