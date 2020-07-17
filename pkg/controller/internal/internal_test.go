package internal

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestErrInstanceNotReady_Error(t *testing.T) {
	var e ErrInstanceNotReady

	assert.Equal(t, ErrInstanceNotReady.Error(e), e.Error())
}

func TestErrRegistryNotReady_Error(t *testing.T) {
	var e ErrRegistryNotReady

	assert.Equal(t, ErrRegistryNotReady.Error(e), e.Error())
}

func TestErrInstanceNotFound_Error(t *testing.T) {
	var e ErrInstanceNotFound

	assert.Equal(t, ErrInstanceNotFound.Error(e), e.Error())
}
