package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	ProjectStatusPhaseName string
	MemberRole             string
	MemberRoleID           int64
)

const (
	MemberRoleIDDefault MemberRoleID = iota
	MemberRoleIDProjectAdmin
	MemberRoleIDDeveloper
	MemberRoleIDGuest
	MemberRoleIDMaster
)

const (
	ProjectStatusPhaseUnknown     ProjectStatusPhaseName = ""
	ProjectStatusPhaseCreating    ProjectStatusPhaseName = "Creating"
	ProjectStatusPhaseReady       ProjectStatusPhaseName = "Ready"
	ProjectStatusPhaseTerminating ProjectStatusPhaseName = "Terminating"

	MemberRoleProjectAdmin MemberRole = "ProjectAdmin"
	MemberRoleDeveloper    MemberRole = "Developer"
	MemberRoleGuest        MemberRole = "Guest"
	MemberRoleMaster       MemberRole = "Master"
)

type ProjectSpec struct {
	Name string `json:"name"`

	// ParentInstance is a LocalObjectReference to the
	// name of the harbor instance the project is created for
	ParentInstance corev1.LocalObjectReference `json:"parentInstance"`

	// +kubebuilder:validation:Optional
	StorageLimit int `json:"storageLimit"`

	// +kubebuilder:validation:Optional
	Metadata ProjectMetadata `json:"metadata,omitempty"`

	// Ref to the name of a 'User' resource
	// +kubebuilder:validation:Optional
	MemberRequests []MemberRequest `json:"memberRequests,omitempty"`
}

type ProjectMetadata struct {
	// Whether content trust is enabled or not
	// If it is, users can not pull unsigned images from this project
	// +kubebuilder:validation:Optional
	EnableContentTrust bool `json:"enableContentTrust,omitempty"`
	// Whether to scan images automatically when pushing or not
	// +kubebuilder:validation:Optional
	AutoScan bool `json:"autoScan,omitempty"`
	// Whether this project reuses the system level CVE allowlist as the allowlist of its own.
	// The valid values are "true", "false".
	// If set to "true", the actual allowlist associated with this project, if any, will be ignored.
	// +kubebuilder:validation:Optional
	ReuseSysCVEAllowlist bool `json:"reuseSysCVEAllowlist,omitempty"`
	// Whether to prevent the vulnerable images from running or not.
	// +kubebuilder:validation:Optional
	PreventVul bool `json:"preventVul,omitempty"`
	// Public status of the Project
	// +kubebuilder:validation:Required
	Public bool `json:"public"`
	// If a vulnerability's severity is higher than the severity defined here,
	// images can't be pulled. Valid values are "none", "low", "medium", "high", "critical".
	// +kubebuilder:validation:Optional
	Severity *string `json:"severity,omitempty"`
}

type MemberRequest struct {
	Role MemberRole                  `json:"role"`
	User corev1.LocalObjectReference `json:"user"` // reference to an User object
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Project is the Schema for the projects API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=project,scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="phase"
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id",description="harbor replication id"
// +kubebuilder:printcolumn:name="Public",type="boolean",JSONPath=".spec.metadata.public",description="harbor replication id"
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ProjectList contains a list of Projects.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

// ProjectStatus defines the state of a single project
type ProjectStatus struct {
	Name    string                 `json:"name"`
	Phase   ProjectStatusPhaseName `json:"phase"`
	Message string                 `json:"message"`
	// Time of last observed transition into this state
	// +kubebuilder:validation:Optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`

	// The project ID is written back from the held project ID.
	ID int32 `json:"id,omitempty"`
	// Members is the list of existing project member users as LocalObjectReference
	Members []corev1.LocalObjectReference `json:"members,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
