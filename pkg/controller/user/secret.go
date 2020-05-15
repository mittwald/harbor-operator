package user

import (
	"context"

	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ReconcileUser) getSecretForUser(ctx context.Context, user *registriesv1alpha1.User) (*corev1.Secret, error) {
	sec := &corev1.Secret{}
	sec.Name = user.Spec.ParentInstance.Name + "-" + user.Spec.UserSecretRef.Name
	sec.Namespace = user.Namespace
	err := r.client.Get(ctx, types.NamespacedName{Name: sec.Name, Namespace: sec.Namespace}, sec)
	if err != nil {
		return &corev1.Secret{}, err
	}
	return sec, nil
}

func (r *ReconcileUser) newSecretForUser(ctx context.Context, user *registriesv1alpha1.User) (*corev1.Secret, error) {
	ls := r.labelsForUserSecret(user, user.Spec.ParentInstance.Name)

	sec := &corev1.Secret{}
	sec.Name = user.Spec.ParentInstance.Name + "-" + user.Spec.UserSecretRef.Name
	sec.Namespace = user.Namespace

	err := r.client.Get(ctx, types.NamespacedName{Name: sec.Name, Namespace: sec.Namespace}, sec)
	if errors.IsNotFound(err) {
		pw, err := helper.NewRandomPassword(16)
		if err != nil {
			return &corev1.Secret{}, err
		}
		sec = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      user.Spec.ParentInstance.Name + "-" + user.Spec.UserSecretRef.Name,
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
				"username": []byte(user.Name),
				"password": []byte(pw),
			},
		}
		return sec, nil
	} else if err != nil {
		return &corev1.Secret{}, err
	}
	return sec, nil
}

func (r *ReconcileUser) getOrCreateSecretForUser(ctx context.Context, user *registriesv1alpha1.User) (*corev1.Secret, error) {
	sec, err := r.getSecretForUser(ctx, user)
	if errors.IsNotFound(err) {
		sec, eerr := r.newSecretForUser(ctx, user)
		if eerr != nil {
			return &corev1.Secret{}, eerr
		}

		err := r.client.Create(ctx, sec)
		if err != nil {
			return &corev1.Secret{}, err
		}
	} else if err != nil {
		return &corev1.Secret{}, err
	}

	return sec, nil
}
