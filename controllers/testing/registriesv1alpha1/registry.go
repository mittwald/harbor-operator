package registriesv1alpha1_test

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRegistry returns a registry object with sample values.
func CreateRegistry(name, namespace, instanceRef string) registriesv1alpha1.Registry {
	r := registriesv1alpha1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: registriesv1alpha1.RegistrySpec{
			Name:           "test-registry",
			Description:    "test registry",
			Type:           "manual",
			URL:            "https://core.harbor.domain",
			Credential:     nil,
			Insecure:       false,
			ParentInstance: corev1.LocalObjectReference{Name: instanceRef},
		},
		Status: registriesv1alpha1.RegistryStatus{ID: 1},
	}

	return r
}
