package v1alpha2

// ID returns a role ID integer by enumerating the given role.
func (role MemberRole) ID() MemberRoleID {
	switch role {
	case MemberRoleProjectAdmin:
		return MemberRoleIDProjectAdmin

	case MemberRoleDeveloper:
		return MemberRoleIDDeveloper

	case MemberRoleGuest:
		return MemberRoleIDGuest

	case MemberRoleMaster:
		return MemberRoleIDMaster
	}

	return MemberRoleIDDefault
}
