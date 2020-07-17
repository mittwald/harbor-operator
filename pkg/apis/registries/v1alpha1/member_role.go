package v1alpha1

// ID returns a role ID integer by enumerating the given role
func (role MemberRole) ID() int64 {
	switch role {
	case MemberRoleProjectAdmin:
		return 1

	case MemberRoleDeveloper:
		return 2

	case MemberRoleGuest:
		return 3

	case MemberRoleMaster:
		return 4
	}

	return 0
}
