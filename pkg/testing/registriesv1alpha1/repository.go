package registriesv1alpha1

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateRepository(name, namespace, instanceRef string) registriesv1alpha1.Repository {
	r := registriesv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: registriesv1alpha1.RepositorySpec{
			Name:           name,
			ParentInstance: corev1.LocalObjectReference{Name: instanceRef},
			ProjectID:      0,
			OwnerID:        nil,
			Deleted:        false,
			Toggleable:     false,
			RoleID:         0,
			CVEWhitelist:   registriesv1alpha1.CVEWhitelist{},
			Metadata:       registriesv1alpha1.RepositoryMetadata{},
			MemberRequests: nil,
		},
		Status: registriesv1alpha1.RepositoryStatus{},
	}

	return r
}
