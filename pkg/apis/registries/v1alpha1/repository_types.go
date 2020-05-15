package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RepositoryStatusPhaseName string
type MemberRole string

const (
	RepositoryStatusPhaseUnknown     RepositoryStatusPhaseName = ""
	RepositoryStatusPhaseCreating    RepositoryStatusPhaseName = "Creating"
	RepositoryStatusPhaseReady       RepositoryStatusPhaseName = "Ready"
	RepositoryStatusPhaseTerminating RepositoryStatusPhaseName = "Terminating"

	MemberRoleProjectAdmin MemberRole = "ProjectAdmin"
	MemberRoleDeveloper    MemberRole = "Developer"
	MemberRoleGuest        MemberRole = "Guest"
	MemberRoleMaster       MemberRole = "Master"
)

type RepositorySpec struct {
	Name string `json:"name"`

	// ParentInstance is a LocalObjectReference to the
	// name of the harbor instance the repository is created for
	ParentInstance corev1.LocalObjectReference `json:"parentInstance"`

	Metadata RepositoryMetadata `json:"metadata"`

	// Ref to the name of a 'User' resource
	// +optional
	MemberRequests []MemberRequest `json:"memberRequests,omitempty"`
}

type RepositoryMetadata struct {
	// Whether content trust is enabled or not
	// If it is, users can not pull unsigned images from this repository
	EnableContentTrust bool `json:"enableContentTrust"`
	// Whether to scan images automatically when pushing or not
	AutoScan bool `json:"autoScan"`
	// If a vulnerability's severity is higher than the severity defined here, images can't be pulled. Valid values are "none", "low", "medium", "high", "critical".
	Severity string `json:"severity"`
	// Whether this repository reuses the system level CVE whitelist as the whitelist of its own. The valid values are "true", "false". If it is set to "true" the actual whitelist associate with this repository, if any, will be ignored.
	ReuseSysSVEWhitelist bool `json:"reuseSysSVEWhitelist"`
	// Whether to prevent the vulnerable images from running or not. The valid values are "true", "false".
	PreventVul bool `json:"preventVul"`
	// Public status of the Repository
	Public bool `json:"public"`
}

type MemberRequest struct {
	Role MemberRole              `json:"role"`
	User v1.LocalObjectReference `json:"user"` // reference to an User object
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Repository is the Schema for the projects API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=repository,scope=Namespaced,shortName=repo;repos;harborrepos
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="phase"
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RepositorySpec `json:"spec,omitempty"`

	Status RepositoryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RepositoryList contains a list of Repository
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

// RepositoryStatus defines the state of a single repository
type RepositoryStatus struct {
	Name    string                    `json:"name"`
	Phase   RepositoryStatusPhaseName `json:"phase"`
	Message string                    `json:"message"`
	// Time of last observed transition into this state
	// +optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}
