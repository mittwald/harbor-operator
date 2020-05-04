package internal

import (
	"errors"
	h "github.com/mittwald/goharbor-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
)

var ErrUserNotFound = errors.New("user not found")
var ErrRegistryNotFound = errors.New("registry not found")
var ErrReplicationNotFound = errors.New("replication not found")

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

// GetRegistry
// Wrapper function checking the command issued to the API, returning a custom error
func GetRegistry(harborClient *h.Client, registry *registriesv1alpha1.Registry) (h.Registry, error) {
	reg, err := harborClient.Registries().GetRegistryByID(registry.Spec.ID)
	if err != nil {
		return h.Registry{}, ErrRegistryNotFound
	}
	return reg, nil
}

// GetReplication
func GetReplication(harborClient *h.Client, replication *registriesv1alpha1.Replication) (h.ReplicationPolicy, error) {
	rep, err := harborClient.Replications().GetReplicationPolicyByID(replication.Spec.ID)
	if err != nil {
		return h.ReplicationPolicy{}, ErrReplicationNotFound
	}
	return rep, nil
}

func GetRoleInt(RoleString string) int {
	switch RoleString {
	case "projectAdmin":
		return 1
	case "developer":
		return 2
	case "guest":
		return 3
	case "master":
		return 4
	}
	return 1
}

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
