package models

import "errors"

type PermissionType string

const RoleDefault = 0
const RoleAdmin = -123
const SubRoleAdminPreset = -123

type RoleDomain int

var ErrRolePreset = errors.New("can't edit preset role")

const RoleDomainDiscepto = RoleDomain(-123)

type Role struct {
	ID     int
	Name   string
	Preset bool
	Domain RoleDomain
}
