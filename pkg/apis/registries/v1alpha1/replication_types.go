package v1alpha1

import (
	h "github.com/mittwald/goharbor-client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReplicationStatusPhaseName string

const (
	ReplicationStatusPhaseUnknown     RepositoryStatusPhaseName = ""
	ReplicationStatusPhaseCreating                              = "Creating"
	ReplicationStatusPhaseReady                                 = "Ready"
	ReplicationStatusPhaseTerminating                           = "Terminating"
)

// const definition
const (
	FilterTypeResource h.FilterType = "resource"
	FilterTypeName                  = "name"
	FilterTypeTag                   = "tag"
	FilterTypeLabel                 = "label"

	TriggerTypeManual     h.TriggerType = "manual"
	TriggerTypeScheduled                = "scheduled"
	TriggerTypeEventBased               = "event_based"
)

// ReplicationSpec defines the desired state of Replication
type ReplicationSpec struct {
	ID int64 `json:"id"`

	Name string `json:"name"`

	Deletion bool `json:"deletion"`

	// +optional
	Description string `json:"description,omitempty"`

	// +optional
	Creator string `json:"creator,omitempty"`

	// The destination namespace
	// If left empty, the resource will be but in the same namespace as the source
	// +optional
	DestNamespace string `json:"dest_namespace,omitempty"`

	// Source Registry
	// This uses the registriesv1alpha1 custom Registry specification
	// +optional
	SrcRegistry *RegistrySpec `json:"src_registry,omitempty"`

	// Destination Registry
	// This uses the registriesv1alpha1 custom Registry specification
	// +optional
	DestRegistry *RegistrySpec `json:"dest_registry,omitempty"`

	// Whether to override the resources on the destination registry or not
	Override bool `json:"override"`

	// Whether the policy is enabled or not
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// The replication policy trigger type
	// +optional
	Trigger *Trigger `json:"trigger,omitempty"`

	// The replication policy filter array
	// +optional
	Filters []Filter `json:"filters,omitempty"`

	// Whether to replicate the deletion operation
	// +optional
	ReplicateDeletion bool `json:"replicate_deletion,omitempty"`

	// ParentInstance is a LocalObjectReference to the
	// name of the harbor instance the replication policy gets created for
	ParentInstance corev1.LocalObjectReference `json:"parentInstance"`
}

// ReplicationStatus defines the observed state of Replication
type ReplicationStatus struct {
	Name    string                    `json:"name"`
	Phase   RepositoryStatusPhaseName `json:"phase"`
	Message string                    `json:"message"`
	// Time of last observed transition into this state
	// +optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`
}

// Filter holds the info of a filter
// Use string instead of interface{}, or else CRD generation will fail
type Filter struct {
	Type  h.FilterType `json:"type"`
	Value string       `json:"value"`
}

// Have to use our custom type here, because we cannot DeepCopy the pointer of *h.Trigger
// Trigger holds info for a trigger
type Trigger struct {
	Type     TriggerType      `json:"type"`
	Settings *TriggerSettings `json:"trigger_settings"`
}

// TriggerSettings holds the settings of a trigger
type TriggerSettings struct {
	Cron string `json:"cron"`
}

// TriggerType represents the type of trigger.
type TriggerType string

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Replication is the Schema for the replications API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=replications,scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="phase"
type Replication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ReplicationSpec `json:"spec,omitempty"`

	Status ReplicationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReplicationList contains a list of Replication
type ReplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Replication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Replication{}, &ReplicationList{})
}
