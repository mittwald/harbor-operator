package v1alpha1

import (
	modelv1 "github.com/mittwald/goharbor-client/model/v1_10_0"
)

// ToHarborRegistry returns a Harbor registry constructed from the provided spec.
func (spec *RegistrySpec) ToHarborRegistry(id int64, credential *modelv1.RegistryCredential) *modelv1.Registry {
	return &modelv1.Registry{
		ID:          id,
		Name:        spec.Name,
		Description: spec.Description,
		Type:        string(spec.Type),
		URL:         spec.URL,
		Credential:  credential,
		Insecure:    spec.Insecure,
	}
}
