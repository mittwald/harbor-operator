package helper

import (
	legacymodel "github.com/mittwald/goharbor-client/v4/apiv2/model/legacy"
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
)

// ToHarborRegistry returns a Harbor registry constructed from the provided spec.
func ToHarborRegistry(spec v1alpha2.RegistrySpec, id int64, credential *legacymodel.RegistryCredential) *legacymodel.Registry {
	return &legacymodel.Registry{
		ID:          id,
		Name:        spec.Name,
		Description: spec.Description,
		Type:        string(spec.Type),
		URL:         spec.URL,
		Credential:  credential,
		Insecure:    spec.Insecure,
	}
}
