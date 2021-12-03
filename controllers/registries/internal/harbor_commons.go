package internal

import (
	"context"
	"fmt"

	h "github.com/mittwald/goharbor-client/v5/apiv2"
	"github.com/mittwald/goharbor-client/v5/apiv2/model"
	"sigs.k8s.io/controller-runtime/pkg/client"

	registriesv1alpha2 "github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"
	"github.com/mittwald/harbor-operator/controllers/registries/helper"
)

const FinalizerName = "registries.mittwald.de/finalizer"

func AssertHealthyHarborInstance(ctx context.Context, harborClient *h.RESTClient) error {
	health, err := harborClient.GetHealth(ctx)
	if err != nil {
		return err
	}

	if health.Status != "healthy" {
		unhealthyComponents := GetUnhealthyComponents(health.Components)
		err := fmt.Errorf("unhealthy components: %q", unhealthyComponents)
		return err
	}

	return nil
}

// GetUnhealthyComponents takes a list of components and returns a list of components that are not healthy.
func GetUnhealthyComponents(status []*model.ComponentHealthStatus) []string {
	var unhealthyComponents []string
	for _, c := range status {
		if c.Status != "healthy" {
			unhealthyComponents = append(unhealthyComponents, c.Name)
		}
	}

	return unhealthyComponents
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
