package v1alpha1

import (
	legacymodel "github.com/mittwald/goharbor-client/v2/apiv2/model/legacy"
)

// ToHarborRegistry returns a Harbor registry constructed from the provided spec.
func (spec *RegistrySpec) ToHarborRegistry(id int64, credential *legacymodel.RegistryCredential) *legacymodel.Registry {
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
