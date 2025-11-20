package storage

// COMMENT  почему файл не называется errors.go
import "errors"

var (
	ErrUserExists        = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrAppNotFound       = errors.New("app not found")
	ErrNoPermissionFound = errors.New("no permission found")
)
