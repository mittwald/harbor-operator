package registriesv1alpha1

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateUser returns a user object with sample values
func CreateUser(name, namespace string) registriesv1alpha1.User {
	u := registriesv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: registriesv1alpha1.UserSpec{
			Name:           name,
			ParentInstance: corev1.LocalObjectReference{},
			RealName:       "harbor user",
			Email:          "test@example.com",
			UserSecretRef:  corev1.LocalObjectReference{},
			AdminRole:      false,
		},
	}

	return u
}
