package internal

import (
	"context"

	h "github.com/mittwald/goharbor-client/v4/apiv2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	registriesv1alpha2 "github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"
	"github.com/mittwald/harbor-operator/controllers/registries/helper"
)

const FinalizerName = "registries.mittwald.de/finalizer"

func HarborInstanceIsHealthy(ctx context.Context, harborClient *h.RESTClient) (bool, error) {
	health, err := harborClient.Health(ctx)
	if err != nil {
		return false, err
	}
	if health.Status != "healthy" {
		return false, &controllererrors.ErrInstanceNotHealthy{}
	}

	return true, nil
}

// GetOperationalHarborInstance returns a harbor instance if it exists.
// Returns an error if the instance could not be found or is not in the 'Installed' phase.
func GetOperationalHarborInstance(ctx context.Context, instanceKey client.ObjectKey, cl client.Client) (*registriesv1alpha2.Instance, error) {
	var instance registriesv1alpha2.Instance

	instanceExists, err := helper.ObjExists(ctx, cl, instanceKey.Name, instanceKey.Namespace, &instance)
	if err != nil {
		return nil, err
	}

	if !instanceExists {
		return nil, &controllererrors.ErrInstanceNotFound{}
	}

	if instance.Status.Phase.Name != registriesv1alpha2.InstanceStatusPhaseInstalled {
		return &instance, &controllererrors.ErrInstanceNotInstalled{}
	}

	return &instance, nil
}
