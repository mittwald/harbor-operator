package testing

import (
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateUser returns a user object with sample values.
func CreateUser(name, namespace, instanceRef string) *v1alpha2.User {
	u := v1alpha2.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.UserSpec{
			Name: name,
			ParentInstance: corev1.LocalObjectReference{
				Name: instanceRef,
			},
			RealName:         "harbor user",
			Email:            "test@example.com",
			UserSecretRef:    corev1.LocalObjectReference{},
			SysAdmin:         false,
			PasswordStrength: 8,
		},
	}

	return &u
}
