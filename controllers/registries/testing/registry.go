package testing

import (
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRegistry returns a registry object with sample values.
func CreateRegistry(name, namespace, instanceRef string) *v1alpha2.Registry {
	r := v1alpha2.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.RegistrySpec{
			Name:           "test-registry",
			Description:    "test registry",
			Type:           "manual",
			URL:            "https://core.harbor.domain",
			Credential:     nil,
			Insecure:       false,
			ParentInstance: corev1.LocalObjectReference{Name: instanceRef},
		},
		Status: v1alpha2.RegistryStatus{ID: 1},
	}

	return &r
}
