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
	InstanceStatusPhaseReady       InstanceStatusPhaseName = "Ready"
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
// +kubebuilder:resource:path=instances,scope=Namespaced,shortName=harborinstance;harbor;harb
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase.name",description="phase"
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
	Name    string `json:"name"`
	Version string `json:"version"`
	// cant use the resulting string-type so this is a simple string and will be casted to an OperatorType in the resolver:
	// error: Hit an unsupported type invalid type for invalid type
	Type string `json:"type"`

	// The FQDN name of the host machine
	InstanceHost string `json:"instanceHost"`

	// +optional
	Options *InstanceDeployOptions `json:"options,omitempty"`

	HelmChart *InstanceHelmChartSpec `json:"helmChart"`

	// +optional
	GarbageCollection *GarbageCollection `json:"garbageCollection,omitempty"`
}

// GarbageCollectionReq holds request information for a garbage collection schedule.
type GarbageCollection struct {
	// +optional
	Cron string `json:"cron,omitempty"`

	// +optional
	ScheduleType ScheduleType `json:"scheduleType,omitempty"`
}

type InstanceDeployOptions struct {
	// +optional
	KubeconfigSecret *KubeconfigSecret `json:"kubeconfigSecret"`
}

type KubeconfigSecret struct {
	SecretRef     *corev1.LocalObjectReference `json:"secretRef"`
	KubeconfigKey string                       `json:"kubeconfigKey"`
}

type InstanceHelmChartSpec struct {
	helmclient.ChartSpec `json:",inline"`

	// set additional chart values from secret
	// +optional
	SecretValues *InstanceHelmChartSecretValues `json:"secretValues,omitempty"`
}

type InstanceHelmChartSecretValues struct {
	SecretRef *corev1.LocalObjectReference `json:"secretRef"`
	Key       string                       `json:"key"`
}

// InstanceStatus defines the observed state of Instance.
type InstanceStatus struct {
	Phase InstanceStatusPhase `json:"phase"`
	// +optional
	Version  string `json:"version"`
	SpecHash string `json:"specHash"`
}

type InstanceStatusPhase struct {
	Name InstanceStatusPhaseName `json:"name"`

	Message string `json:"message"`

	// Time of last observed transition into this state.
	// +optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`
}
