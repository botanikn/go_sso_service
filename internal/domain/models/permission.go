package models

// Permission is a user's role within a single app.
// Values must match the permission_type enum in the database.
type Permission string

const (
	PermissionBanned Permission = "banned"
	PermissionUser   Permission = "user"
	PermissionAdmin  Permission = "admin"
)

func (p Permission) IsValid() bool {
	switch p {
	case PermissionBanned, PermissionUser, PermissionAdmin:
		return true
	default:
		return false
	}
}
