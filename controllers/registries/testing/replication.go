package testing

import (
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateReplication returns a replication object with sample values.
func CreateReplication(name, namespace, instanceRef string) *v1alpha2.Replication {
	r := v1alpha2.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.ReplicationSpec{
			Name:              name,
			Description:       "",
			Creator:           "",
			DestNamespace:     "",
			SrcRegistry:       nil,
			DestRegistry:      nil,
			ReplicateDeletion: false,
			ParentInstance:    corev1.LocalObjectReference{Name: instanceRef},
		},
		Status: v1alpha2.ReplicationStatus{ID: 1},
	}

	return &r
}
