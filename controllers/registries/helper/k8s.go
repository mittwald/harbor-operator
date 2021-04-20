package helper

import (
	"context"
	"fmt"

	registriesv1alpha2 "github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjExists returns a boolean value based on the existence of a runtime object.
func ObjExists(ctx context.Context, client client.Client, name, namespace string, obj client.Object) (bool, error) {
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, obj)
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

// GetOperationalHarborInstance returns a goharbor instance, if it exists and is in the 'Installed' phase.
func GetOperationalHarborInstance(ctx context.Context, instanceKey client.ObjectKey, cl client.Client) (*registriesv1alpha2.Instance, error) {
	var instance registriesv1alpha2.Instance

	instanceExists, err := ObjExists(ctx, cl, instanceKey.Name, instanceKey.Namespace, &instance)
	if err != nil {
		return nil, err
	}

	if !instanceExists {
		return nil, &controllererrors.ErrInstanceNotFound{}
	}

	// Don't reconcile if the corresponding harbor instance is not in 'Installed' phase.
	if instance.Status.Phase.Name != registriesv1alpha2.InstanceStatusPhaseInstalled {
		return &instance, &controllererrors.ErrInstanceNotInstalled{}
	}

	return &instance, nil
}
