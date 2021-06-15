package models

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrEmailAlreadyUsed = errors.New("email already used")
	ErrInvalidFormat    = errors.New("invalid email format")
	ErrWeakPasswd       = errors.New("weak password")
)

type User struct {
	ID    int
	Name  string
	Email string
	Bio   string
}

type UserView struct {
	User
	Karma     int
	CreatedAt time.Time
}

type Member struct {
	UserID int
	Name   string
	Roles  []Role
	LeftAt sql.NullTime
}
