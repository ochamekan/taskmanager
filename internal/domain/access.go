package domain

var errNotMember = Forbidden("you are not a member of this team")

func (m Membership) CanViewTeam() error {
	if !m.IsMember {
		return errNotMember
	}
	return nil
}

func (m Membership) CanCreateTask() error { return m.CanViewTeam() }

func (m Membership) CanComment() error { return m.CanViewTeam() }

func (m Membership) CanViewStats() error {
	if !m.IsMember {
		return errNotMember
	}
	if !m.isManager() {
		return Forbidden("team analytics is available to the owner and admins only")
	}
	return nil
}

func (m Membership) CanInvite(newRole Role) error {
	if !m.IsMember {
		return errNotMember
	}
	if !m.isManager() {
		return Forbidden("only the team owner or an admin can invite users")
	}
	switch newRole {
	case RoleMember:
		return nil
	case RoleAdmin:
		if m.Role != RoleOwner {
			return Forbidden("only the team owner can grant the admin role")
		}
		return nil
	case RoleOwner:
		return Forbidden("the owner role cannot be granted by invitation")
	default:
		return Validation("role must be one of: admin, member")
	}
}

func (m Membership) CanUpdateTask(task Task, changedFields []string) error {
	if !m.IsMember {
		return errNotMember
	}
	if m.isManager() {
		return nil
	}

	isCreator := task.CreatedBy == m.UserID
	isAssignee := task.AssigneeID != nil && *task.AssigneeID == m.UserID

	switch {
	case isCreator:
		for _, field := range changedFields {
			switch field {
			case FieldTitle, FieldDescription, FieldStatus, FieldAssigneeID:
			default:
				return Forbidden("field " + field + " cannot be changed")
			}
		}
		return nil
	case isAssignee:
		for _, field := range changedFields {
			if field != FieldStatus {
				return Forbidden("the assignee can only change the task status")
			}
		}
		return nil
	default:
		return Forbidden("you can only edit tasks you created or are assigned to")
	}
}
