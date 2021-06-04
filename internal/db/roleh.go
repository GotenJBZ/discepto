package db

import (
	"context"

	"gitlab.com/ranfdev/discepto/internal/models"
)

type RoleH struct {
	id       int
	name     string
	preset   bool
	domain   models.RoleDomain
	sharedDB DBTX
}

func (h *RoleH) ListActivePerms(ctx context.Context) (models.Perms, error) {
	return listRolePerms(ctx, h.sharedDB, h.id)
}
func (h *RoleH) UpdatePerms(ctx context.Context, perms models.Perms) error {
	if h.preset {
		return models.ErrRolePreset
	}
	return setPermissions(ctx, h.sharedDB, h.id, perms)
}
func (h *RoleH) CanEdit() bool {
	return !h.preset
}
func (h *RoleH) DeleteRole(ctx context.Context) error {
	if h.preset {
		return models.ErrPermDenied
	}
	return deleteRole(ctx, h.sharedDB, h.id)
}
