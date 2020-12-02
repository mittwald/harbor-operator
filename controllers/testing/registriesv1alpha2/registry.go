package registriesv1alpha2_test

import (
	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRegistry returns a registry object with sample values.
func CreateRegistry(name, namespace, instanceRef string) *registriesv1alpha2.Registry {
	r := registriesv1alpha2.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: registriesv1alpha2.RegistrySpec{
			Name:           "test-registry",
			Description:    "test registry",
			Type:           "manual",
			URL:            "https://core.harbor.domain",
			Credential:     nil,
			Insecure:       false,
			ParentInstance: corev1.LocalObjectReference{Name: instanceRef},
		},
		Status: registriesv1alpha2.RegistryStatus{ID: 1},
	}

	return &r
}
