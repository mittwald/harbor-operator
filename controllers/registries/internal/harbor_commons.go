package internal

import (
	"context"

	h "github.com/mittwald/goharbor-client/v4/apiv2"
	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"
)

const FinalizerName = "harbor-operator.registries.mittwald.de"

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
