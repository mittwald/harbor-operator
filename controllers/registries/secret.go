package registries

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	"github.com/mittwald/harbor-operator/controllers/registries/helper"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (r *UserReconciler) getSecretForUser(ctx context.Context, user *v1alpha2.User) (*corev1.Secret, error) {
	sec := &corev1.Secret{}

	err := r.Client.Get(ctx, client.ObjectKey{
		Name: user.Spec.UserSecretRef.Name, Namespace: user.Namespace,
	}, sec)
	if err != nil {
		return &corev1.Secret{}, err
	}

	return sec, nil
}

func (r *UserReconciler) newSecretForUser(ctx context.Context, user *v1alpha2.User) (*corev1.Secret, error) {
	sec := &corev1.Secret{}

	exists, err := helper.ObjExists(ctx, r.Client, user.Spec.UserSecretRef.Name, user.Namespace, sec)
	if err != nil {
		return nil, err
	}
	if !exists {
		pw, err := helper.NewRandomPassword(user.Spec.PasswordStrength)
		if err != nil {
			return nil, err
		}

		sec = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      user.Spec.UserSecretRef.Name,
				Namespace: user.Namespace,
				Labels:    r.labelsForUserSecret(user, user.Spec.ParentInstance.Name),
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: user.APIVersion,
					Kind:       user.Kind,
					Name:       user.Name,
					UID:        user.UID,
				}},
			},
			Data: map[string][]byte{
				"username": []byte(user.Spec.Name),
				"password": []byte(pw),
			},
		}
		return sec, nil
	}

	return nil, fmt.Errorf("could not create or get user secret")
}

func (r *UserReconciler) getOrCreateSecretForUser(ctx context.Context,
	user *v1alpha2.User) (*corev1.Secret, error) {
	sec, err := r.getSecretForUser(ctx, user)
	if err != nil {
		if errors.IsNotFound(err) {
			sec, err = r.newSecretForUser(ctx, user)
			if err != nil {
				return nil, err
			}

			err := r.Client.Create(ctx, sec)
			if err != nil {
				return nil, err
			}

			createdSecret := corev1.Secret{}

			return &createdSecret, r.Client.Get(ctx, client.ObjectKey{
				Name:      user.Spec.UserSecretRef.Name,
				Namespace: user.Namespace,
			}, &createdSecret)
		}

		return nil, err
	}

	err = controllerutil.SetControllerReference(user, sec, r.Scheme)
	if err != nil {
		return nil, err
	}

	return sec, nil
}
