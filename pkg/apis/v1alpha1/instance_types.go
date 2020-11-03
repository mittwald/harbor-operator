package v1alpha1

import (
	helmclient "github.com/mittwald/go-helm-client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstanceStatusPhaseName string

const (
	InstanceStatusPhasePending     InstanceStatusPhaseName = "Pending"
	InstanceStatusPhaseInstalling  InstanceStatusPhaseName = "Installing"
	InstanceStatusPhaseInstalled   InstanceStatusPhaseName = "Installed"
	InstanceStatusPhaseTerminating InstanceStatusPhaseName = "Terminating"
	InstanceStatusPhaseError       InstanceStatusPhaseName = "Error"
)

type ScheduleType string

const (
	ScheduleTypeHourly   ScheduleType = "Hourly"
	ScheduleTypeDaily    ScheduleType = "Daily"
	ScheduleTypeWeekly   ScheduleType = "Weekly"
	ScheduleTypeCustom   ScheduleType = "Custom"
	ScheduleTypeManually ScheduleType = "Manually"
	ScheduleTypeNone     ScheduleType = "None"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Instance is the Schema for the instances API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=instances,scope=Namespaced,shortName=harborinstance;harbor
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase.name",description="phase"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version",description="instance version"
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.instanceURL", description="harbor instance url"
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InstanceSpec `json:"spec,omitempty"`

	Status InstanceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// InstanceList contains a list of Instance.
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instance{}, &InstanceList{})
}

// InstanceSpec defines the desired state of Instance.
type InstanceSpec struct {
	Name string `json:"name"`
	// can't use the resulting string-type so this is a simple string and will be casted to an OperatorType in the resolver:
	// error: Hit an unsupported type invalid type for invalid type
	Type string `json:"type"`

	InstanceURL string `json:"instanceURL"`

	HelmChart *InstanceHelmChartSpec `json:"helmChart"`

	// +kubebuilder:validation:Optional
	GarbageCollection *GarbageCollection `json:"garbageCollection,omitempty"`
}

// GarbageCollectionReq holds request information for a garbage collection schedule.
type GarbageCollection struct {
	// +kubebuilder:validation:Optional
	Cron string `json:"cron,omitempty"`

	// +kubebuilder:validation:Optional
	ScheduleType ScheduleType `json:"scheduleType,omitempty"`
}

type InstanceHelmChartSpec struct {
	helmclient.ChartSpec `json:",inline"`

	// set additional chart values from secret
	// +kubebuilder:validation:Optional
	SecretValues *InstanceHelmChartSecretValues `json:"secretValues,omitempty"`
}

type InstanceHelmChartSecretValues struct {
	SecretRef *corev1.LocalObjectReference `json:"secretRef"`
	Key       string                       `json:"key"`
}

// InstanceStatus defines the observed state of Instance.
type InstanceStatus struct {
	Phase InstanceStatusPhase `json:"phase"`

	// +kubebuilder:validation:Optional
	Version string `json:"version"`

	SpecHash string `json:"specHash"`
}

type InstanceStatusPhase struct {
	Name InstanceStatusPhaseName `json:"name"`

	Message string `json:"message"`

	// Time of last observed transition into this state.
	// +kubebuilder:validation:Optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`
}
