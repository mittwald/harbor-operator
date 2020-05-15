package registriesv1alpha1

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateReplication returns a replication object with sample values
func CreateReplication(name, namespace, instanceRef string) registriesv1alpha1.Replication {
	r := registriesv1alpha1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: registriesv1alpha1.ReplicationSpec{
			ID:            1,
			Name:          name,
			Deletion:      false,
			Description:   "",
			Creator:       "",
			DestNamespace: "",
			// These are intentionally left nil and should be specified in the individual tests
			SrcRegistry:       nil,
			DestRegistry:      nil,
			ReplicateDeletion: false,
			ParentInstance:    corev1.LocalObjectReference{Name: instanceRef},
		},
	}

	return r
}
