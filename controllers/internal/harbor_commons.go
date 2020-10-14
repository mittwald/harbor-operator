package internal

import (
	"context"
	"fmt"

	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "harbor-operator.registries.mittwald.de"

// ErrInstanceNotFound is called when the corresponding Harbor instance could not be found.
type ErrInstanceNotFound string

// ErrInstanceNotFound is called when the corresponding Harbor instance is not ready.
type ErrInstanceNotReady string

// ErrRegistryNotReady is called when the corresponding RegistryCR (registriesv1alpha1.Registry) is not ready.
type ErrRegistryNotReady string

func (e ErrInstanceNotFound) Error() string {
	return fmt.Sprintf("instance '%s' not found", string(e))
}

func (e ErrInstanceNotReady) Error() string {
	return fmt.Sprintf("instance '%s' not ready", string(e))
}

func (e ErrRegistryNotReady) Error() string {
	return fmt.Sprintf("registry '%s' not ready", string(e))
}

// FetchReadyHarborInstance returns a harbor instance based on the provided instance name
// Also needs a controller client to fetch the actual instance.
func FetchReadyHarborInstance(ctx context.Context, namespace, parentInstanceName string,
	r client.Client) (*registriesv1alpha1.Instance, error) {
	harbor := &registriesv1alpha1.Instance{}
	ns := types.NamespacedName{
		Namespace: namespace,
		Name:      parentInstanceName,
	}

	err := r.Get(ctx, ns, harbor)
	if apierrors.IsNotFound(err) {
		return nil, ErrInstanceNotFound(parentInstanceName)
	} else if err != nil {
		return nil, err
	}

	// Reconcile only if the corresponding harbor instance is in 'Ready' state
	if harbor.Status.Phase.Name != registriesv1alpha1.InstanceStatusPhaseInstalled {
		return nil, ErrInstanceNotReady(parentInstanceName)
	}

	return harbor, nil
}
