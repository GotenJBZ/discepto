package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID    int
	Name  string
	Email string
	Bio   string
}

type UserView struct {
	User
	Karma int
	CreatedAt time.Time
}

type Member struct {
	UserID int
	Name   string
	Roles  []Role
	LeftAt sql.NullTime
}
