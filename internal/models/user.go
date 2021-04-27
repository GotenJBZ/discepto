package models

import "database/sql"

type User struct {
	ID    int
	Name  string
	Email string
}

type Member struct {
	UserID int
	Name   string
	Roles  []Role
	LeftAt sql.NullTime
}
