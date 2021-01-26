package testing

import (
	"github.com/mittwald/goharbor-client/v3/apiv2/retention"
	registriesv1alpha2 "github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRetention returns a retention object with sample values.
func CreateRetention(name, namespace, projectRef, instanceRef string) *registriesv1alpha2.Retention {
	r := registriesv1alpha2.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: registriesv1alpha2.RetentionSpec{
			Name:              name,
			Algorithm:         retention.AlgorithmOr,
			Scope:             registriesv1alpha2.RetentionPolicyScope{},
			Trigger:           registriesv1alpha2.RetentionRuleTrigger{},
			ProjectReferences: []corev1.LocalObjectReference{{Name: projectRef}},
			Rules:             []registriesv1alpha2.RetentionRule{},
			ParentInstance:    corev1.LocalObjectReference{Name: instanceRef},
		},
		Status: registriesv1alpha2.RetentionStatus{},
	}

	return &r
}
