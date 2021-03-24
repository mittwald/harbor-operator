package internal

import (
	"context"
	"strings"

	h "github.com/mittwald/goharbor-client/v3/apiv2"
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"
)

const FinalizerName = "harbor-operator.registries.mittwald.de"

func HarborInstanceIsHealthy(ctx context.Context, harborClient *h.RESTClient, harbor *v1alpha2.Instance) (bool, error) {
	// Don't reconcile if the corresponding harbor instance is not in 'Installed' phase.
	if harbor.Status.Phase.Name != v1alpha2.InstanceStatusPhaseInstalled {
		return false,
			controllererrors.ErrInstanceNotInstalled(
				strings.Join([]string{harbor.Name, harbor.Namespace}, "/"))
	}

	health, err := harborClient.Health(ctx)
	if err != nil {
		return false, err
	}
	if health.Status != "healthy" {
		return false, controllererrors.ErrInstanceNotHealthy(
			strings.Join([]string{harbor.Name, harbor.Namespace}, "/"))
	}

	return true, nil
}
