package registriesv1alpha1

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRepository returns a repository object with sample values.
func CreateRepository(name, namespace, instanceRef string) registriesv1alpha1.Repository {
	r := registriesv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: registriesv1alpha1.RepositorySpec{
			Name:           name,
			ParentInstance: corev1.LocalObjectReference{Name: instanceRef},
			Metadata:       registriesv1alpha1.RepositoryMetadata{},
			MemberRequests: nil,
		},
		Status: registriesv1alpha1.RepositoryStatus{},
	}

	return r
}
