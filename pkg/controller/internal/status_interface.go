package internal

import registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"

type SetStatus interface {
	SetRegistryStatus(registry *registriesv1alpha1.Registry, status registriesv1alpha1.RegistryStatus) error
}
