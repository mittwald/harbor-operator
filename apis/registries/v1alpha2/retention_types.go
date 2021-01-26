package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	RetentionStatusPhaseName string
)

const (
	RetentionStatusPhaseUnknown     = ""
	RetentionStatusPhaseCreating    = "Creating"
	RetentionStatusPhaseActive      = "Active"
	RetentionStatusPhaseTerminating = "Terminating"

	// AlgorithmOr is the default algorithm when operating on harbor retention rules
	AlgorithmOr string = "or"

	// Key for defining matching repositories
	ScopeSelectorRepoMatches = "repoMatches"

	// Key for defining excluded repositories
	ScopeSelectorRepoExcludes = "repoExcludes"

	// Key for defining matching tag expressions
	TagSelectorMatches = "matches"

	// Key for defining excluded tag expressions
	TagSelectorExcludes = "excludes"

	// The kind of the retention selector, _always_ defaults to 'doublestar'
	SelectorTypeDefault = "doublestar"

	// Retain the most recently pushed n artifacts - count
	PolicyTemplateLatestPushedArtifacts = "latestPushedK"

	// Retain the most recently pulled n artifacts - count
	PolicyTemplateLatestPulledArtifacts = "latestPulledN"

	// Retain the artifacts pushed within the last n days
	PolicyTemplateDaysSinceLastPush = "nDaysSinceLastPush"

	// Retain the artifacts pulled within the last n days
	PolicyTemplateDaysSinceLastPull = "nDaysSinceLastPull"

	// Retain always
	PolicyTemplateRetainAlways = "always"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Retention is the Schema for the retentions API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=retention,scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="phase"
type Retention struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RetentionSpec   `json:"spec,omitempty"`
	Status RetentionStatus `json:"status,omitempty"`
}

type RetentionSpec struct {
	Name string `json:"name"`

	// The algorithm used for the retention policy.
	// +kubebuilder:validation:Optional
	Algorithm string `json:"algorithm,omitempty"`

	Scope RetentionPolicyScope `json:"scope,omitempty"`

	//
	Trigger RetentionRuleTrigger `json:"trigger,omitempty"`

	// ProjectReferences is a a list of LocalObjectReferences to individual project resources.
	// Retentions are created for each project contained in the list.
	// If it is empty, however, no operation is performed.
	ProjectReferences []corev1.LocalObjectReference `json:"projectReferences,omitempty"`

	// A list of retention rules to a maximum of 15 items.
	// See https://goharbor.io/docs/2.1.0/working-with-projects/working-with-images/create-tag-retention-rules/
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxItems:=15
	Rules []RetentionRule `json:"rules,omitempty"`

	// ParentInstance is a LocalObjectReference to the
	// name of the harbor instance the Retention is created for.
	ParentInstance corev1.LocalObjectReference `json:"parentInstance"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RetentionList contains a list of Retentions.
type RetentionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Retention `json:"items"`
}

type RetentionRuleTrigger struct {
	// +kubebuilder:validation:Optional
	Kind string `json:"kind,omitempty"`

	// +kubebuilder:validation:Optional
	References string `json:"references,omitempty"`

	// +kubebuilder:validation:Optional
	Settings string `json:"settings,omitempty"`
}

type RetentionPolicyScope struct {
	Level string `json:"level,omitempty"`

	Ref int64 `json:"ref,omitempty"`
}

type RetentionRule struct {
	// action
	Action string `json:"action,omitempty"`

	// disabled
	Disabled bool `json:"disabled,omitempty"`

	// id TODO omitting this, as it seems not to be required (apparently set to 0 when adding additional rules)
	// ID int64 `json:"id,omitempty"`

	// TODO: One of the valid params has to be provided "latestPushedK" etc.
	// params
	Params map[string]string `json:"params,omitempty"`

	// priority
	Priority int64 `json:"priority,omitempty"`

	// scope selectors
	ScopeSelectors map[string][]string `json:"scope_selectors,omitempty"`

	// tag selectors
	TagSelectors []*string `json:"tag_selectors"`

	// template
	Template string `json:"template,omitempty"`
}

type RetentionSelector struct {
	// decoration
	Decoration string `json:"decoration,omitempty"`

	// extras
	Extras string `json:"extras,omitempty"`

	// kind
	Kind string `json:"kind,omitempty"`

	// pattern
	Pattern string `json:"pattern,omitempty"`
}

// ProjectStatus defines the state of a single project
type RetentionStatus struct {
	Name    string                   `json:"name"`
	Phase   RetentionStatusPhaseName `json:"phase"`
	Message string                   `json:"message"`
	// Time of last observed transition into this state
	// +kubebuilder:validation:Optional
	LastTransition *metav1.Time `json:"lastTransition,omitempty"`

	// ProjectReference is a list of ProjectReferences
	ProjectReferences []ProjectReference `json:"projectReferences,omitempty"`
}

// ProjectReference presents the observed meta information about
// a single project that this retention has been created for.
type ProjectReference struct {
	Name        string `json:"name,omitempty"`
	ProjectID   int32  `json:"projectID,omitempty"`
	RetentionID int32  `json:"retentionID,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Retention{}, &RetentionList{})
}
