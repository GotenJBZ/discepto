package models

type User struct {
	ID     int
	Name   string
	Email  string
	RoleID int `db:"role_id"`
}
