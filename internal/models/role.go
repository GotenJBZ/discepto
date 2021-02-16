package models

import "time"

type PermissionType string

const RoleDefault = 0
const RoleAdmin = -123

const (
	PermissionTypeAddMods     PermissionType = "add_mods"
	PermissionTypeDeletePosts PermissionType = "delete_posts"
	PermissionTypeBanUsers    PermissionType = "ban_users"
	PermissionTypeFlagPosts   PermissionType = "flag_posts"
)

type RoleID int

type Role struct {
	ID          int
	Name        string
	Permissions string
	Origin      int
	CreatedAt   time.Time
}
