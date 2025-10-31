package entity

import "time"

type User struct {
	ID    string
	Email string
	Name  string
	Password string
	IsAdmin bool
	Token string
	CreatedAt time.Time
	UpdatedAt time.Time
}
