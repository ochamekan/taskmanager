package domain

import (
	"errors"
	"testing"
)

const (
	ownerID    = 1
	adminID    = 2
	creatorID  = 3
	assigneeID = 4
	otherID    = 5
	outsiderID = 6
)

func member(id int64, r Role) Membership {
	return Membership{UserID: id, IsMember: true, Role: r}
}

var (
	owner    = member(ownerID, RoleOwner)
	admin    = member(adminID, RoleAdmin)
	creator  = member(creatorID, RoleMember)
	assignee = member(assigneeID, RoleMember)
	other    = member(otherID, RoleMember)
	outsider = Membership{UserID: outsiderID}
)

func ptr[T any](v T) *T { return &v }

func task() Task {
	return Task{ID: 10, TeamID: 1, CreatedBy: creatorID, AssigneeID: ptr(int64(assigneeID)), Status: StatusTodo}
}

func expect(t *testing.T, name string, err error, kind error) {
	t.Helper()
	if kind == nil {
		if err != nil {
			t.Errorf("%s: expected allowed, got %v", name, err)
		}
		return
	}
	if !errors.Is(err, kind) {
		t.Errorf("%s: expected %v, got %v", name, kind, err)
	}
}

func TestCanUpdateTask(t *testing.T) {
	all := []string{FieldTitle, FieldDescription, FieldStatus, FieldAssigneeID}
	cases := []struct {
		name    string
		actor   Membership
		changed []string
		want    error
	}{
		{"owner edits anything", owner, all, nil},
		{"admin edits anything", admin, all, nil},
		{"creator edits own task fully", creator, all, nil},
		{"creator reassigns", creator, []string{FieldAssigneeID}, nil},
		{"assignee changes status", assignee, []string{FieldStatus}, nil},
		{"assignee cannot reassign", assignee, []string{FieldAssigneeID}, ErrForbidden},
		{"assignee cannot retitle", assignee, []string{FieldTitle}, ErrForbidden},
		{"assignee cannot mix status and description", assignee, []string{FieldStatus, FieldDescription}, ErrForbidden},
		{"assignee no-op update", assignee, nil, nil},
		{"unrelated member cannot edit", other, []string{FieldStatus}, ErrForbidden},
		{"unrelated member cannot even no-op", other, nil, ErrForbidden},
		{"outsider cannot edit", outsider, []string{FieldStatus}, ErrForbidden},
		{"outsider with owner role field is still outsider", Membership{UserID: outsiderID, Role: RoleOwner}, all, ErrForbidden},
	}
	for _, c := range cases {
		expect(t, c.name, c.actor.CanUpdateTask(task(), c.changed), c.want)
	}
}

func TestCanUpdateTask_CreatorWhoIsAlsoAssignee(t *testing.T) {
	tk := task()
	tk.AssigneeID = ptr(int64(creatorID))
	expect(t, "creator+assignee retitles", creator.CanUpdateTask(tk, []string{FieldTitle}), nil)
}

func TestCanUpdateTask_UnassignedTask(t *testing.T) {
	tk := task()
	tk.AssigneeID = nil
	expect(t, "former assignee loses rights", assignee.CanUpdateTask(tk, []string{FieldStatus}), ErrForbidden)
}

func TestCanInvite(t *testing.T) {
	cases := []struct {
		name  string
		actor Membership
		role  Role
		want  error
	}{
		{"owner invites member", owner, RoleMember, nil},
		{"owner invites admin", owner, RoleAdmin, nil},
		{"owner cannot grant owner", owner, RoleOwner, ErrForbidden},
		{"admin invites member", admin, RoleMember, nil},
		{"admin cannot grant admin", admin, RoleAdmin, ErrForbidden},
		{"admin cannot grant owner", admin, RoleOwner, ErrForbidden},
		{"member cannot invite", other, RoleMember, ErrForbidden},
		{"outsider cannot invite", outsider, RoleMember, ErrForbidden},
		{"unknown role", owner, Role("superuser"), ErrValidation},
	}
	for _, c := range cases {
		expect(t, c.name, c.actor.CanInvite(c.role), c.want)
	}
}

func TestVisibilityAndStats(t *testing.T) {
	expect(t, "member views team", other.CanViewTeam(), nil)
	expect(t, "outsider cannot view team", outsider.CanViewTeam(), ErrForbidden)
	expect(t, "member creates task", other.CanCreateTask(), nil)
	expect(t, "outsider cannot create task", outsider.CanCreateTask(), ErrForbidden)
	expect(t, "member comments", other.CanComment(), nil)
	expect(t, "outsider cannot comment", outsider.CanComment(), ErrForbidden)

	expect(t, "owner sees stats", owner.CanViewStats(), nil)
	expect(t, "admin sees stats", admin.CanViewStats(), nil)
	expect(t, "member cannot see stats", other.CanViewStats(), ErrForbidden)
	expect(t, "outsider cannot see stats", outsider.CanViewStats(), ErrForbidden)
}
