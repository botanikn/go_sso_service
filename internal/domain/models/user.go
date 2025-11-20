package models

type User struct {
	ID       string
	Username string
	Email    string
	PassHash []byte
}
