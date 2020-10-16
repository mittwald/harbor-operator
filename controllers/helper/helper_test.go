package helper_test

import (
	"context"
	"testing"

	"github.com/mittwald/harbor-operator/controllers/helper"

	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/harbor-operator/controllers/internal/mocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestBoolToString(t *testing.T) {
	assert.Equal(t, helper.BoolToString(false), "false")
	assert.Equal(t, helper.BoolToString(true), "true")
}

func TestGetValueFromSecret(t *testing.T) {
	sec := &corev1.Secret{
		Data: map[string][]byte{
			"a": []byte("b"),
		},
	}

	t.Run("ExistingKey", func(t *testing.T) {
		val, err := helper.GetValueFromSecret(sec, "a")

		if assert.NoError(t, err) {
			assert.Equal(t, val, "b")
		}
	})

	t.Run("NonExistentKey", func(t *testing.T) {
		_, err := helper.GetValueFromSecret(sec, "z")

		if assert.NotNil(t, err) {
			assert.Errorf(t, err,
				"could not find key %s in secret %s, namespace %s", "z", sec.Name, sec.Namespace)
		}
	})
}

func TestInterfaceHash(t *testing.T) {
	var i helper.InterfaceHash = make([]byte, 16)

	t.Run("Short", func(t *testing.T) {
		short := i.Short()

		if assert.NotNil(t, short) {
			assert.Equal(t, len(short), 8)
		}
	})

	t.Run("String", func(t *testing.T) {
		str := i.String()

		if assert.NotNil(t, str) {
			assert.IsType(t, "", str)
		}
	})
}

func TestJSONPatch_AddOp(t *testing.T) {
	var p helper.JSONPatch

	p.AddOp("foo", "bar", "baz")
	assert.Equal(t, 1, len(p.Ops))
}

func TestJSONPatch_Data(t *testing.T) {
	var p helper.JSONPatch

	sec := &corev1.Secret{}

	p.AddOp("foo", "bar", "baz")

	b, err := p.Data(sec)
	assert.NoError(t, err)
	assert.IsType(t, b, []byte{})
}

func TestJSONPatch_Type(t *testing.T) {
	var p helper.JSONPatch

	j := p.Type()

	assert.IsType(t, j, types.JSONPatchType)
}

func TestNewRandomPassword(t *testing.T) {
	pw, err := helper.NewRandomPassword(8)

	assert.NoError(t, err)

	assert.Equal(t, 8, len(pw))
}

func TestCreateSpecHash(t *testing.T) {
	spec := &helmclient.ChartSpec{}

	hash, err := helper.CreateSpecHash(spec)

	assert.NoError(t, err)

	hash2, err := helper.CreateSpecHash(spec)

	assert.NoError(t, err)

	if assert.NotNil(t, hash) && assert.NotNil(t, hash2) {
		assert.Equal(t, hash, hash2)
	}

	spec.ChartName = "foo"

	hash3, err := helper.CreateSpecHash(spec)

	assert.NoError(t, err)

	if assert.NotNil(t, hash3) {
		assert.NotEqual(t, hash, hash3)
	}
}

func TestPushFinalizer(t *testing.T) {
	o := &corev1.Pod{}
	finalizer := "foo"
	// Push finalizer twice to cover already existing finalizers
	helper.PushFinalizer(o, finalizer)
	helper.PushFinalizer(o, finalizer)
}

func TestPullFinalizer(t *testing.T) {
	finalizer := "foo"
	finalizer2 := "bar"
	o := &corev1.Pod{}

	t.Run("existing finalizer", func(t *testing.T) {
		// Add the finalizer before pulling
		helper.PushFinalizer(o, finalizer)
		helper.PullFinalizer(o, finalizer)
	})

	t.Run("non existent finalizer", func(t *testing.T) {
		helper.PushFinalizer(o, finalizer)
		helper.PullFinalizer(o, finalizer2)
	})
}

func TestObjExists(t *testing.T) {
	ctx := context.TODO()
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
		},
	}

	mockClient := &mocks.MockClient{}

	t.Run("ExistingObject", func(t *testing.T) {
		mockClient.On("Get", ctx, types.NamespacedName{
			Namespace: sec.Namespace,
			Name:      sec.Name,
		}, sec).Return(nil)

		exists, err := helper.ObjExists(ctx, mockClient, sec.Name, sec.Namespace, sec)

		assert.NoError(t, err)
		assert.Equal(t, exists, true)
		mockClient.AssertExpectations(t)
	})
}
