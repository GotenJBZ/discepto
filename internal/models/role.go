package models

type PermissionType string

const (
	PermissionTypeAddMods     PermissionType = "add_mods"
	PermissionTypeDeletePosts PermissionType = "delete_posts"
	PermissionTypeBanUsers    PermissionType = "ban_users"
	PermissionTypeFlagPosts   PermissionType = "flag_posts"
)

type RoleID int

const RoleAdmin = 1

type Role struct {
	ID          int
	Name        string
	Permissions []RolePermission
}
type RolePermission struct {
	RoleID     int
	Permission PermissionType
}
