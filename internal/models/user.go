package models

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrEmailAlreadyUsed = errors.New("Email already used")
	ErrInvalidFormat    = errors.New("Invalid email format")
	ErrWeakPasswd       = errors.New("Weak password")
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
