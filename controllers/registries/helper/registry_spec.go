package helper

import (
	"github.com/mittwald/goharbor-client/v5/apiv2/model"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
)

// ToHarborRegistry returns a Harbor registry constructed from the provided spec.
func ToHarborRegistry(spec v1alpha2.RegistrySpec, id int64, credential *model.RegistryCredential) *model.Registry {
	return &model.Registry{
		ID:          id,
		Name:        spec.Name,
		Description: spec.Description,
		Type:        string(spec.Type),
		URL:         spec.URL,
		Credential:  credential,
		Insecure:    spec.Insecure,
	}
}
