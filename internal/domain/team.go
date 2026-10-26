package domain

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type Team struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	CreatedBy       int64     `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	CurrentUserRole Role      `json:"role,omitempty"`
}

type Membership struct {
	UserID   int64
	IsMember bool
	Role     Role
}

func (m Membership) isManager() bool {
	return m.IsMember && (m.Role == RoleOwner || m.Role == RoleAdmin)
}
