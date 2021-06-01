package db

import (
	"context"

	"gitlab.com/ranfdev/discepto/internal/models"
)

type RolePerms struct {
	ManageRole bool
	UpdateRole bool
	DeleteRole bool
}
type RoleH struct {
	id        int
	name      string
	domain    models.RoleDomain
	rolePerms RolePerms
	sharedDB  DBTX
}

func (h *RoleH) ListActivePerms(ctx context.Context) (map[string]bool, error) {
	return listRolePerms(ctx, h.sharedDB, h.id)
}
func (h *RoleH) UpdatePerms(ctx context.Context, perms map[string]bool) error {
	if !h.rolePerms.UpdateRole {
		return models.ErrPermDenied
	}
	return setPermissions(ctx, h.sharedDB, h.id, perms)
}
func (h *RoleH) Perms() RolePerms {
	return h.rolePerms
}
func (h *RoleH) DeleteRole(ctx context.Context) error {
	if !h.rolePerms.DeleteRole {
		return models.ErrPermDenied
	}
	return deleteRole(ctx, h.sharedDB, h.id)
}
