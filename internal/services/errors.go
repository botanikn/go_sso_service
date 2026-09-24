package services

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidArgument is the sentinel behind ValidationError.
	ErrInvalidArgument        = errors.New("invalid argument")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrUserAlreadyExists      = errors.New("user already exists")
	ErrUserBanned             = errors.New("user is banned")
	ErrAppDoesNotExist        = errors.New("app does not exist")
	ErrPermissionDoesNotExist = errors.New("permission does not exist")
	ErrInvalidToken           = errors.New("invalid token")
	ErrTokenExpired           = errors.New("token expired")
	ErrForbidden              = errors.New("insufficient permissions")
)

// ValidationError describes invalid client input. Its message is safe to return to clients.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return ErrInvalidArgument.Error() + ": " + e.Msg }

func (e *ValidationError) Unwrap() error { return ErrInvalidArgument }

func InvalidArgument(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}
