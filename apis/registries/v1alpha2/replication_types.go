package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReplicationStatusPhaseName string

const (
	ReplicationStatusPhaseUnknown                 ReplicationStatusPhaseName = ""
	ReplicationStatusPhaseCreating                ReplicationStatusPhaseName = "Creating"
	ReplicationStatusPhaseCompleted               ReplicationStatusPhaseName = "Completed"
	ReplicationStatusPhaseTerminating             ReplicationStatusPhaseName = "Terminating"
	ReplicationStatusPhaseManualExecutionRunning  ReplicationStatusPhaseName = "Execution Running"
	ReplicationStatusPhaseManualExecutionFailed   ReplicationStatusPhaseName = "Execution Failed"
	ReplicationStatusPhaseManualExecutionFinished ReplicationStatusPhaseName = "Execution Finished"
)

// const definition
const (
	ReplicationTriggerTypeManual     string = "manual"
	ReplicationTriggerTypeScheduled  string = "scheduled"
	ReplicationTriggerTypeEventBased string = "event_based"

	ReplicationFilterTypeResource string = "resource"
	ReplicationFilterTypeName     string = "name"
	ReplicationFilterTypeTag      string = "tag"
	ReplicationFilterTypeLabel    string = "label"
)

// ReplicationSpec defines the desired state of Replication
type ReplicationSpec struct {
	// Whether to override the resources on the destination registry or not
	Override bool `json:"override"`

	// Whether the policy is enabled or not
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:Optional
	TriggerAfterCreation bool `json:"triggerAfterCreation,omitempty"`
	// Whether to replicate the deletion operation
	// +kubebuilder:validation:Optional
	ReplicateDeletion bool `json:"replicateDeletion,omitempty"`

	// The name of the replication
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Optional
	Creator string `json:"creator,omitempty"`

	// The destination namespace
	// If left empty, the resource will be but in the same namespace as the source
	// +kubebuilder:validation:Optional
	DestNamespace string `json:"destNamespace,omitempty"`

	// Source Registry
	// Reference to a registry cr
	// +kubebuilder:validation:Optional
	SrcRegistry *corev1.LocalObjectReference `json:"srcRegistry,omitempty"`

	// Destination Registry
	// Reference to a registry cr
	// +kubebuilder:validation:Optional
	DestRegistry *corev1.LocalObjectReference `json:"destRegistry,omitempty"`

	// The replication policy trigger type
	// +kubebuilder:validation:Optional
	Trigger *ReplicationTrigger `json:"trigger,omitempty"`

	// The replication policy filter array
	// +kubebuilder:validation:Optional
	Filters []ReplicationFilter `json:"filters,omitempty"`

	// ParentInstance is a LocalObjectReference to the
	// name of the harbor instance the replication policy gets created for
	ParentInstance corev1.LocalObjectReference `json:"parentInstance"`
}

// ReplicationStatus defines the observed state of Replication
type ReplicationStatus struct {
	Phase   ReplicationStatusPhaseName `json:"phase"`
	Message string                     `json:"message"`
	// Time of last observed transition into this state
	// +kubebuilder:validation:Optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`

	// The replication ID is written back from the held replication ID.
	ID int64 `json:"id,omitempty"`
	// The respective source and destination registries
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination,omitempty"`
}

// ReplicationTrigger defines a replication trigger.
// We have to use our custom type here, because we cannot DeepCopy the pointer of *h.Trigger.
type ReplicationTrigger struct {
	// +kubebuilder:validation:Optional
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	Settings *TriggerSettings `json:"triggerSettings,omitempty"`
}

// ReplicationFilter holds the specifications of a replication filter
type ReplicationFilter struct {
	// The replication policy filter type.
	// +kubebuilder:validation:Optional
	Type string `json:"type,omitempty"`

	// The value of replication policy filter.
	// +kubebuilder:validation:Optional
	Value string `json:"value,omitempty"`
}

// TriggerSettings holds the settings of a trigger
type TriggerSettings struct {
	Cron string `json:"cron"`
}

// Replication is the Schema for the replications API
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=replications,scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="phase"
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id",description="harbor replication id"
// +kubebuilder:printcolumn:name="Enabled",type="boolean",JSONPath=".spec.enabled",description="harbor replication id"
// +kubebuilder:printcolumn:name="Source",type="string",JSONPath=".status.source",description="source registry"
// +kubebuilder:printcolumn:name="Destination",type="string",JSONPath=".status.destination",description="destination registry"
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
