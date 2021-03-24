package errors

import "fmt"

// ErrInstanceNotFound is called when the corresponding Harbor instance could not be found.
type ErrInstanceNotFound string

// ErrInstanceNotInstalled is called when the corresponding Harbor instance is not yet installed properly.
type ErrInstanceNotInstalled string

// ErrInstanceNotHealthy is called when the corresponding Harbor instance is not yet reporting healthy.
type ErrInstanceNotHealthy string

// ErrRegistryNotReady is called when the corresponding RegistryCR (registries.Registry) is not ready.
type ErrRegistryNotReady string

func (e ErrInstanceNotFound) Error() string {
	return fmt.Sprintf("instance '%s' not found", string(e))
}

func (e ErrInstanceNotInstalled) Error() string {
	return fmt.Sprintf("instance '%s' is not yet installed", string(e))
}

func (e ErrRegistryNotReady) Error() string {
	return fmt.Sprintf("registry '%s' not ready", string(e))
}

func (e ErrInstanceNotHealthy) Error() string {
	return fmt.Sprintf("instance '%s' is not yet healthy", string(e))
}
