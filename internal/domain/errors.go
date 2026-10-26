package domain

import "errors"

var (
	ErrValidation   = errors.New("validation error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Kind }

func Validation(msg string) *Error   { return &Error{Kind: ErrValidation, Message: msg} }
func Unauthorized(msg string) *Error { return &Error{Kind: ErrUnauthorized, Message: msg} }
func Forbidden(msg string) *Error    { return &Error{Kind: ErrForbidden, Message: msg} }
func NotFound(msg string) *Error     { return &Error{Kind: ErrNotFound, Message: msg} }
func Conflict(msg string) *Error     { return &Error{Kind: ErrConflict, Message: msg} }
