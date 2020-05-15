package v1alpha1

func (role MemberRole) ID() int {
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
