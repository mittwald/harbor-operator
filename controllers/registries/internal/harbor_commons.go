package internal

import (
	"context"
	"fmt"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "harbor-operator.registries.mittwald.de"

// ErrInstanceNotFound is called when the corresponding Harbor instance could not be found.
type ErrInstanceNotFound string

// ErrInstanceNotFound is called when the corresponding Harbor instance is not ready.
type ErrInstanceNotReady string

// ErrRegistryNotReady is called when the corresponding RegistryCR (registries.Registry) is not ready.
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
	r client.Client) (*v1alpha2.Instance, error) {
	harbor := &v1alpha2.Instance{}
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
	if harbor.Status.Phase.Name != v1alpha2.InstanceStatusPhaseInstalled {
		return nil, ErrInstanceNotReady(parentInstanceName)
	}

	return harbor, nil
}
