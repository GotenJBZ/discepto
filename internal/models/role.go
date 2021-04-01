package models

import "time"

type PermissionType string

const RoleDefault = 0
const RoleAdmin = -123

type RoleID int

type Role struct {
	ID          int
	Name        string
	Permissions string
	Origin      int
	CreatedAt   time.Time
}
