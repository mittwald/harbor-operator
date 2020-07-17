package v1alpha1

import (
	modelv1 "github.com/mittwald/goharbor-client/model/v1_10_0"
)

// ToHarborRegistry returns a Harbor registry constructed from the provided spec
func (spec *RegistrySpec) ToHarborRegistry() *modelv1.Registry {
	return &modelv1.Registry{
		ID:          spec.ID,
		Name:        spec.Name,
		Description: spec.Description,
		Type:        spec.Type,
		URL:         spec.URL,
		Credential:  spec.Credential,
		Insecure:    spec.Insecure,
	}
}
