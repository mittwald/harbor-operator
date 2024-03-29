package helper

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjExists returns a boolean value based on the existence of a runtime object.
func ObjExists(ctx context.Context, cl client.Client, name, namespace string, obj client.Object) (bool, error) {
	err := cl.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// GetValueFromSecret returns a specific value of a secret key.
func GetValueFromSecret(sec *corev1.Secret, key string) (string, error) {
	val, ok := sec.Data[key]
	if !ok {
		return "", fmt.Errorf("could not find key %s in secret %s, namespace %s", key, sec.Name, sec.Namespace)
	}

	return string(val), nil
}
