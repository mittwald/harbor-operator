package registriesv1alpha1_test

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRepository returns a repository object with sample values.
func CreateRepository(name, namespace, instanceRef string) registriesv1alpha1.Project {
	r := registriesv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: registriesv1alpha1.ProjectSpec{
			Name:           name,
			ParentInstance: corev1.LocalObjectReference{Name: instanceRef},
			Metadata:       registriesv1alpha1.ProjectMetadata{},
			MemberRequests: nil,
		},
		Status: registriesv1alpha1.ProjectStatus{},
	}

	return r
}
