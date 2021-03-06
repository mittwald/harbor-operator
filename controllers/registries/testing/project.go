package testing

import (
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateProject returns a project object with sample values.
func CreateProject(name, namespace, instanceRef string) *v1alpha2.Project {
	r := v1alpha2.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.ProjectSpec{
			Name:           name,
			ParentInstance: corev1.LocalObjectReference{Name: instanceRef},
			Metadata:       v1alpha2.ProjectMetadata{},
			MemberRequests: nil,
		},
		Status: v1alpha2.ProjectStatus{},
	}

	return &r
}
