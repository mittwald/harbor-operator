package errors

const (
	ErrInstanceNotFoundMsg     = "instance not found"
	ErrInstanceNotInstalledMsg = "instance is not installed"
	ErrInstanceNotHealthyMsg   = "instance is not healthy"
	ErrRegistryNotReadyMsg     = "instance is not ready"
)

// ErrInstanceNotFound is called when the corresponding Harbor instance could not be found.
type ErrInstanceNotFound struct{}

func (e *ErrInstanceNotFound) Error() string {
	return ErrInstanceNotFoundMsg
}

// ErrInstanceNotInstalled is called when the corresponding Harbor instance is not yet installed properly.
type ErrInstanceNotInstalled struct{}

func (e *ErrInstanceNotInstalled) Error() string {
	return ErrInstanceNotInstalledMsg
}

// ErrInstanceNotHealthy is called when the corresponding Harbor instance is not yet reporting healthy.
type ErrInstanceNotHealthy struct{}

func (e *ErrInstanceNotHealthy) Error() string {
	return ErrInstanceNotHealthyMsg
}

// ErrRegistryNotReady is called when the corresponding RegistryCR (registries.Registry) is not ready.
type ErrRegistryNotReady struct{}

func (e *ErrRegistryNotReady) Error() string {
	return ErrRegistryNotReadyMsg
}
