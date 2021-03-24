package internal

import (
	"context"
	"testing"

	registriestesting "github.com/mittwald/harbor-operator/controllers/registries/testing"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const ns = "test-namespace"

func TestBuildClient(t *testing.T) {
	ctx := context.TODO()

	fakeClient := fake.NewClientBuilder().Build()

	harbor := registriestesting.CreateInstance("test-harbor", ns)
	_ = registriestesting.CreateSecret(harbor.Spec.Name+"-harbor-core", ns)

	harborClient, err := BuildClient(ctx, fakeClient, harbor)

	assert.Nil(t, harborClient)
	if assert.Error(t, err) {
		assert.Errorf(t, err, "could not find key HARBOR_ADMIN_PASSWORD in secret , namespace")
	}
}
