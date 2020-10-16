package controllers

import (
	"context"
	"reflect"

	"github.com/mittwald/harbor-operator/controllers/helper"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *UserReconciler) getSecretForUser(ctx context.Context, user *registriesv1alpha1.User) (*corev1.Secret, error) {
	sec := &corev1.Secret{}
	sec.Name = user.Spec.UserSecretRef.Name
	sec.Namespace = user.Namespace

	err := r.Client.Get(ctx, types.NamespacedName{Name: sec.Name, Namespace: sec.Namespace}, sec)
	if err != nil {
		return &corev1.Secret{}, err
	}

	return sec, nil
}

func (r *UserReconciler) newSecretForUser(ctx context.Context, user *registriesv1alpha1.User) (*corev1.Secret, error) {
	ls := r.labelsForUserSecret(user, user.Spec.ParentInstance.Name)

	sec := &corev1.Secret{}
	sec.Name = user.Spec.UserSecretRef.Name
	sec.Namespace = user.Namespace

	err := r.Client.Get(ctx, types.NamespacedName{Name: sec.Name, Namespace: sec.Namespace}, sec)
	if errors.IsNotFound(err) {
		pw, err := helper.NewRandomPassword(user.Spec.PasswordStrength)
		if err != nil {
			return &corev1.Secret{}, err
		}

		sec = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      user.Spec.UserSecretRef.Name,
				Namespace: user.Namespace,
				Labels:    ls,
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
	} else if err != nil {
		return &corev1.Secret{}, err
	}

	return sec, nil
}

func (r *UserReconciler) getOrCreateSecretForUser(ctx context.Context,
	user *registriesv1alpha1.User) (*corev1.Secret, error) {
	sec, err := r.getSecretForUser(ctx, user)
	if errors.IsNotFound(err) {
		sec, err = r.newSecretForUser(ctx, user)
		if err != nil {
			return nil, err
		}

		if err := r.Client.Create(ctx, sec); err != nil {
			return &corev1.Secret{}, err
		}
	} else if err != nil {
		return nil, err
	}

	originalSec := sec.DeepCopy()

	err = controllerutil.SetControllerReference(user, sec, r.Scheme)
	if err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(originalSec, sec) {
		err = r.Client.Update(ctx, sec)
		if err != nil {
			return nil, err
		}
	}

	return sec, nil
}
