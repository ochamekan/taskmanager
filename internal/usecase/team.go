package usecase

import (
	"context"
	"errors"
	"net/http"

	"github.com/example/taskmanager/internal/domain"
)

type createTeamRequest struct {
	Name string `json:"name"`
}

func (u *Usecase) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req createTeamRequest
	if err := decodeJSON(r, &req); err != nil {
		u.writeError(w, r, err)
		return
	}
	name, err := requiredText("name", req.Name, maxNameLength)
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	actorID := currentUserID(r)
	team := &domain.Team{Name: name, CreatedBy: actorID, CreatedAt: nowUTC(), CurrentUserRole: domain.RoleOwner}
	err = u.storage.InTx(r.Context(), func(tx Storage) error {
		if err := tx.CreateTeam(r.Context(), team); err != nil {
			return err
		}
		return tx.AddTeamMember(r.Context(), team.ID, actorID, domain.RoleOwner, team.CreatedAt)
	})
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, team)
}

func (u *Usecase) ListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := u.storage.ListTeamsForUser(r.Context(), currentUserID(r))
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": teams})
}

type inviteRequest struct {
	UserID *int64      `json:"user_id"`
	Email  string      `json:"email"`
	Role   domain.Role `json:"role"`
}

func (u *Usecase) Invite(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathID(r, "id")
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	var req inviteRequest
	if err := decodeJSON(r, &req); err != nil {
		u.writeError(w, r, err)
		return
	}
	if req.Role == "" {
		req.Role = domain.RoleMember
	}
	if req.UserID == nil && req.Email == "" {
		u.writeError(w, r, domain.Validation("user_id or email is required"))
		return
	}
	if req.UserID != nil && *req.UserID <= 0 {
		u.writeError(w, r, domain.Validation("user_id must be a positive integer"))
		return
	}

	actor, err := u.storage.GetMembership(r.Context(), teamID, currentUserID(r))
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	if err := actor.CanInvite(req.Role); err != nil {
		u.writeError(w, r, err)
		return
	}

	user, err := u.findInvitedUser(r.Context(), req)
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	joinedAt := nowUTC()
	err = u.storage.AddTeamMember(r.Context(), teamID, user.ID, req.Role, joinedAt)
	if errors.Is(err, ErrDuplicateRecord) {
		u.writeError(w, r, domain.Conflict("user is already a member of this team"))
		return
	}
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"team_id":   teamID,
		"user_id":   user.ID,
		"role":      req.Role,
		"name":      user.Name,
		"email":     user.Email,
		"joined_at": joinedAt,
	})
}

func (u *Usecase) findInvitedUser(ctx context.Context, req inviteRequest) (*domain.User, error) {
	var user *domain.User
	var err error
	if req.UserID != nil {
		user, err = u.storage.GetUserByID(ctx, *req.UserID)
	} else {
		user, err = u.storage.GetUserByEmail(ctx, normalizeEmail(req.Email))
	}
	if errors.Is(err, ErrRecordNotFound) {
		return nil, domain.NotFound("user not found")
	}
	return user, err
}
