package usecase

import (
	"errors"
	"net/http"

	"github.com/example/taskmanager/internal/auth"
	"github.com/example/taskmanager/internal/domain"
)

var errBadCredentials = domain.Unauthorized("invalid email or password")

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (u *Usecase) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		u.writeError(w, r, err)
		return
	}
	email, err := validEmail(req.Email)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	name, err := requiredText("name", req.Name, maxNameLength)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	if err := validPassword(req.Password); err != nil {
		u.writeError(w, r, err)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	user := &domain.User{Email: email, Name: name, PasswordHash: hash, CreatedAt: nowUTC()}

	err = u.storage.CreateUser(r.Context(), user)
	if errors.Is(err, ErrDuplicateRecord) {
		u.writeError(w, r, domain.Conflict("a user with this email already exists"))
		return
	}
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (u *Usecase) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		u.writeError(w, r, err)
		return
	}

	user, err := u.storage.GetUserByEmail(r.Context(), normalizeEmail(req.Email))
	if errors.Is(err, ErrRecordNotFound) {
		u.writeError(w, r, errBadCredentials)
		return
	}
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		u.writeError(w, r, errBadCredentials)
		return
	}

	token, expiresAt, err := auth.NewToken(user.ID, u.jwtSecret, u.jwtTTL)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_at":   expiresAt,
		"user":         user,
	})
}
