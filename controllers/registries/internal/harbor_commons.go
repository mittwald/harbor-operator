package internal

import (
	"context"

	h "github.com/mittwald/goharbor-client/v4/apiv2"

	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"
)

// deprecated: Use FinalizerName instead. See [^1]
// [^1]: https://sdk.operatorframework.io/docs/upgrading-sdk-version/v1.4.0/#change-your-operators-finalizer-names
const OldFinalizerName = "harbor-operator.registries.mittwald.de"

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
