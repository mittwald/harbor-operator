package internal

import (
	"context"
	"errors"
	"fmt"

	h "github.com/mittwald/goharbor-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrInstanceNotFound is called when the corresponding Harbor instance could not be found
type ErrInstanceNotFound string

// ErrInstanceNotFound is called when the corresponding Harbor instance is not ready
type ErrInstanceNotReady string

// ErrRegistryNotReady is called when the corresponding RegistryCR (registriesv1alpha1.Registry) is not ready
type ErrRegistryNotReady string

func (e ErrInstanceNotFound) Error() string {
	return fmt.Sprintf("instance '%s' not found", string(e))
}

func (e ErrInstanceNotReady) Error() string {
	return fmt.Sprintf("instance '%s' not ready", string(e))
}

func (e ErrRegistryNotReady) Error() string {
	return fmt.Sprintf("registry '%s' not ready", string(e))
}

// ErrUserNotFound is a custom error type describing the absence of a user
var ErrUserNotFound = errors.New("user not found")

// ErrRegistryNotFound is a custom error type describing the absence of a registry
var ErrRegistryNotFound = errors.New("registry not found")

// ErrReplicationNotFound is a custom error type describing the absence of a replication
var ErrReplicationNotFound = errors.New("replication not found")

// GetUser returns a Harbor user, filtering the results of a user search query for the specified user
func GetUser(user *registriesv1alpha1.User, harborClient *h.Client) (h.User, error) {
	// Need to get the user's id (which is determined by harbor itself) to reliably get the user's ID
	resUsers, err := harborClient.Users().SearchUser(h.UserMember{
		Username: user.Spec.Name,
	})
	if err != nil {
		return h.User{}, err
	}

	if len(resUsers) == 0 {
		return h.User{}, ErrUserNotFound
	}

	// uSearchReq now holds a UserSearchRequest with a Username + UserID
	// which we can use to query and get the whole user profile from /users/{user_id}
	uReq := h.UserRequest{
		Username: resUsers[0].Username,
		UserID:   resUsers[0].UserID,
	}

	return harborClient.Users().GetUser(uReq)
}

// GetRegistry gets and returns a registry object
func GetRegistry(harborClient *h.Client, registry *registriesv1alpha1.Registry) (h.Registry, error) {
	if registry.Spec.ID != 0 {
		reg, err := harborClient.Registries().GetRegistryByID(registry.Spec.ID)
		if err != nil {
			return h.Registry{}, ErrRegistryNotFound
		}

		return reg, nil
	}

	reg, err := harborClient.Registries().GetRegistryByName(registry.Spec.Name)
	if err != nil {
		return h.Registry{}, ErrRegistryNotFound
	}

	return reg, nil
}

// GetReplication gets and returns a replication object
func GetReplication(harborClient *h.Client, replication *registriesv1alpha1.Replication) (h.ReplicationPolicy, error) {
	if replication.Spec.ID != 0 {
		rep, err := harborClient.Replications().GetReplicationPolicyByID(replication.Spec.ID)
		if err != nil {
			return h.ReplicationPolicy{}, ErrReplicationNotFound
		}

		return rep, nil
	}

	rep, err := harborClient.Replications().GetReplicationPolicyByName(replication.Spec.Name)
	if err != nil {
		return h.ReplicationPolicy{}, ErrReplicationNotFound
	}

	return rep, nil
}

// CheckAndGetReplicationTriggerType enumerates the specified trigger type and returns a trigger type used by Harbor
func CheckAndGetReplicationTriggerType(providedType registriesv1alpha1.TriggerType) (h.TriggerType, error) {
	switch providedType {
	case "event_based":
		return registriesv1alpha1.TriggerTypeEventBased, nil
	case "manual":
		return registriesv1alpha1.TriggerTypeManual, nil
	case "scheduled":
		return registriesv1alpha1.TriggerTypeScheduled, nil
	}
	return "", errors.New("the provided trigger type could not be validated")
}

// CheckFilterType enumerates the specified filter type
func CheckFilterType(filterType h.FilterType) error {
	switch filterType {
	case registriesv1alpha1.FilterTypeLabel:
		return nil
	case registriesv1alpha1.FilterTypeName:
		return nil
	case registriesv1alpha1.FilterTypeResource:
		return nil
	case registriesv1alpha1.FilterTypeTag:
		return nil
	}
	return errors.New("the provided filter type could not be checked")
}

// FetchReadyHarborInstance returns a harbor instance based on the provided instance name
// Also needs a controller client to fetch the actual instance
func FetchReadyHarborInstance(ctx context.Context, namespace, parentInstanceName string, r client.Client) (*registriesv1alpha1.Instance, error) {
	harbor := &registriesv1alpha1.Instance{}
	ns := types.NamespacedName{
		Namespace: namespace,
		Name:      parentInstanceName,
	}

	err := r.Get(ctx, ns, harbor)
	if apierrors.IsNotFound(err) {
		return nil, ErrInstanceNotFound(parentInstanceName)
	} else if err != nil {
		return nil, err
	}

	// Reconcile only if the corresponding harbor instance is in 'Ready' state
	if harbor.Status.Phase.Name != registriesv1alpha1.InstanceStatusPhaseReady {
		return nil, ErrInstanceNotReady(parentInstanceName)
	}
	return harbor, nil
}
