package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type UserStatusPhaseName string

const (
	UserStatusPhaseUnknown     UserStatusPhaseName = ""
	UserStatusPhaseCreating    UserStatusPhaseName = "Creating"
	UserStatusPhaseReady       UserStatusPhaseName = "Ready"
	UserStatusPhaseTerminating UserStatusPhaseName = "Terminating"
)

type UserSpec struct {
	Name string `json:"name"`

	// ParentInstance is a LocalObjectReference to the
	// name of the harbor instance the user is created for
	ParentInstance corev1.LocalObjectReference `json:"parentInstance"`
	RealName       string                      `json:"realname"`
	Email          string                      `json:"email"`
	UserSecretRef  corev1.LocalObjectReference `json:"userSecretRef"`
	AdminRole      bool                        `json:"adminRole"`
	// +optional
	Comments string `json:"comments,omitempty"`
	// The effective length of the generated user password
	PasswordStrength int32 `json:"passwordStrength"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// User is the Schema for the users API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=users,scope=Namespaced,shortName=users;harborusers
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="phase"
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              UserSpec `json:"spec,omitempty"`

	Status UserStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

// UserStatus defines the state of a single user
type UserStatus struct {
	Name         string              `json:"name"`
	Phase        UserStatusPhaseName `json:"phase"`
	Message      string              `json:"message"`
	PasswordHash string              `json:"passwordHash"`

	// Time of last observed transition into this state
	// +optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
