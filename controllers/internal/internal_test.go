package internal

import (
	"context"
	"testing"

	"github.com/mittwald/harbor-operator/controllers/internal/mocks"
	testingregistriesv1alpha1 "github.com/mittwald/harbor-operator/controllers/testing/registriesv1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const ns = "test-namespace"

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

func TestBuildClient(t *testing.T) {
	ctx := context.TODO()

	mockClient := &mocks.MockClient{}

	harbor := testingregistriesv1alpha1.CreateInstance("test-harbor", ns)
	_ = testingregistriesv1alpha1.CreateSecret(harbor.Spec.Name+"-harbor-core", ns)
	// sec := &corev1.Secret{}

	t.Run("SecretKeyNotFound", func(t *testing.T) {
		mockClient.On("Get", ctx, types.NamespacedName{
			Namespace: ns,
			Name:      harbor.Spec.Name + "-harbor-core",
		}, &corev1.Secret{}).Return(nil)

		harborClient, err := BuildClient(ctx, mockClient, &harbor)

		assert.Nil(t, harborClient)
		if assert.Error(t, err) {
			assert.Errorf(t, err, "could not find key HARBOR_ADMIN_PASSWORD in secret , namespace")
		}

		mockClient.AssertExpectations(t)
	})
}
