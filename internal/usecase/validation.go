package usecase

import (
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/example/taskmanager/internal/domain"
)

const (
	maxNameLength        = 255
	maxEmailLength       = 255
	maxTitleLength       = 255
	maxDescriptionLength = 10_000
	maxCommentLength     = 5_000
	minPasswordLength    = 8
	maxPasswordBytes     = 72
)

func requiredText(field, value string, maxLength int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", domain.Validation(field + " is required")
	}
	return value, checkLength(field, value, maxLength)
}

func optionalText(field, value string, maxLength int) (string, error) {
	value = strings.TrimSpace(value)
	return value, checkLength(field, value, maxLength)
}

func checkLength(field, value string, maxLength int) error {
	if utf8.RuneCountInString(value) > maxLength {
		return domain.Validation(field + " must be at most " + strconv.Itoa(maxLength) + " characters")
	}
	return nil
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func validEmail(raw string) (string, error) {
	email := normalizeEmail(raw)
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > maxEmailLength {
		return "", domain.Validation("email is invalid")
	}
	return email, nil
}

func validPassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordLength || len(password) > maxPasswordBytes {
		return domain.Validation("password must be at least 8 characters and at most 72 bytes long")
	}
	return nil
}

func validStatus(status domain.TaskStatus) error {
	if !status.Valid() {
		return domain.Validation("status must be one of: todo, in_progress, done")
	}
	return nil
}

type Page struct {
	Limit  int
	Offset int
}

func (p Page) withDefaults(defaultLimit, maxLimit int) (Page, error) {
	if p.Limit == 0 {
		p.Limit = defaultLimit
	}
	if p.Limit < 1 || p.Limit > maxLimit {
		return p, domain.Validation("limit must be between 1 and " + strconv.Itoa(maxLimit))
	}
	if p.Offset < 0 {
		return p, domain.Validation("offset must be >= 0")
	}
	return p, nil
}
