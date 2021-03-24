package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrInstanceNotInstalled_Error(t *testing.T) {
	var e ErrInstanceNotInstalled

	assert.Equal(t, ErrInstanceNotInstalled.Error(e), e.Error())
}

func TestErrRegistryNotReady_Error(t *testing.T) {
	var e ErrRegistryNotReady

	assert.Equal(t, ErrRegistryNotReady.Error(e), e.Error())
}

func TestErrErrInstanceNotHealthy_Error(t *testing.T) {
	var e ErrInstanceNotHealthy

	assert.Equal(t, ErrInstanceNotHealthy.Error(e), e.Error())
}

func TestErrInstanceNotFound_Error(t *testing.T) {
	var e ErrInstanceNotFound

	assert.Equal(t, ErrInstanceNotFound.Error(e), e.Error())
}
