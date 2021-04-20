package helper

import (
	"context"

	h "github.com/mittwald/goharbor-client/v3/apiv2"
)

// AdminUserExists checks whether the preconfigured 'admin' user already exists and returns accordingly.
// The user ID of the administrator account is fixed to '1'.
func AdminUserExists(ctx context.Context, harborClient *h.RESTClient) (bool, error) {
	a, err := harborClient.GetUserByID(ctx, 1)
	if err != nil {
		return false, err
	}
	return a != nil, nil
}
