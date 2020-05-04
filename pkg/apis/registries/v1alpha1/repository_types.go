package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RepositoryStatusPhaseName string

const (
	RepositoryStatusPhaseUnknown     RepositoryStatusPhaseName = ""
	RepositoryStatusPhaseCreating    RepositoryStatusPhaseName = "Creating"
	RepositoryStatusPhaseReady       RepositoryStatusPhaseName = "Ready"
	RepositoryStatusPhaseTerminating RepositoryStatusPhaseName = "Terminating"
)

type RepositorySpec struct {
	Name string `json:"name"`

	// ParentInstance is a LocalObjectReference to the
	// name of the harbor instance the repository is created for
	ParentInstance corev1.LocalObjectReference `json:"parentInstance"`

	// +optional
	ProjectID int64 `json:"projectID,omitempty"`

	// The owner ID of the repository creator
	// +optional
	OwnerID *int `json:"ownerID,omitempty"`

	// +optional
	Deleted bool `json:"deleted,omitempty"`

	// Correspond to the UI about whether the repository's publicity is updatable (for UI)
	Toggleable bool `json:"toggleable"`

	// +optional
	RoleID int `json:"roleID,omitempty"`

	// +optional
	CVEWhitelist CVEWhitelist `json:"CVEWhitelist,omitempty"`

	Metadata RepositoryMetadata `json:"metadata"`

	// Ref to the name of a 'User' resource
	// +optional
	MemberRequests []RepositoryMemberRequest `json:"memberRequests,omitempty"`
}

type RepositoryMemberRequest struct {
	RoleID     int        `json:"roleID"`
	MemberUser MemberUser `json:"memberUser"`
}

type MemberUser struct {
	Username string `json:"username"`
	// +optional
	UserID int `json:"userID,omitempty"`
}

type CVEWhitelistItem struct {
	// +optional
	CVEID string `json:"CVEID,omitempty"`
}

type CVEWhitelist struct {
	ID        int64 `json:"id"`
	ProjectID int64 `json:"projectID"`
	// +optional
	Items []CVEWhitelistItem `orm:"-" json:"items,omitempty"`
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
