package models

import "database/sql"

type User struct {
	ID    int
	Name  string
	Email string
	Bio   string
}

type UserView struct {
	User
	Karma int
}

type Member struct {
	UserID int
	Name   string
	Roles  []Role
	LeftAt sql.NullTime
}
