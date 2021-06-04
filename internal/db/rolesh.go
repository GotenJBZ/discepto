package db

import (
	"context"

	"gitlab.com/ranfdev/discepto/internal/models"
)

type RolesH struct {
	contextPerms models.Perms
	rolesPerms   models.Perms
	domain       models.RoleDomain
	sharedDB     DBTX
}

func newUnsafeRolesH(db DBTX, perms models.Perms, domain models.RoleDomain) *RolesH {
	return &RolesH{
		contextPerms: perms,
		rolesPerms:   models.NewPerms(models.PermManageRole),
		domain:       domain,
		sharedDB:     db,
	}
}

func (h *RolesH) Assign(ctx context.Context, toUser int, roleH RoleH) error {
	if err := h.rolesPerms.Require(models.PermManageRole); err != nil {
		return err
	}
	if roleH.domain != h.domain {
		return models.ErrPermDenied
	}
	newRolePerms, err := roleH.ListActivePerms(ctx)
	if err != nil {
		return err
	}
	if err := h.contextPerms.RequirePerms(newRolePerms); err != nil {
		return err
	}
	return assignRole(ctx, h.sharedDB, toUser, roleH.id)
}

func (h *RolesH) Unassign(ctx context.Context, toUser int, roleH RoleH) error {
	if err := h.rolesPerms.Require(models.PermManageRole); err != nil {
		return err
	}
	if roleH.domain != h.domain {
		return models.ErrPermDenied
	}
	newRolePerms, err := roleH.ListActivePerms(ctx)
	if err != nil {
		return err
	}
	if err := h.contextPerms.RequirePerms(newRolePerms); err != nil {
		return err
	}
	return unassignRole(ctx, h.sharedDB, toUser, roleH.id)
}

func (h *RolesH) ListRoles(ctx context.Context) ([]models.Role, error) {
	if err := h.rolesPerms.Require(models.PermManageRole); err != nil {
		return nil, err
	}

	return listRoles(ctx, h.sharedDB, h.domain)
}

func (h *RolesH) ListUserRoles(ctx context.Context, userID int) ([]models.Role, error) {
	if err := h.rolesPerms.Require(models.PermManageRole); err != nil {
		return nil, err
	}

	return listUserRoles(ctx, h.sharedDB, userID, h.domain)
}

func (h *RolesH) UnassignAll(ctx context.Context, userID int) error {
	if err := h.rolesPerms.Require(models.PermManageRole); err != nil {
		return err
	}
	return unassignAll(ctx, h.sharedDB, userID, h.domain)
}

func (h *RolesH) CreateRole(ctx context.Context, roleName string) (*RoleH, error) {
	if err := h.rolesPerms.Require(models.PermManageRole); err != nil {
		return nil, err
	}
	role := models.Role{
		Domain: h.domain,
		Name:   roleName,
		Preset: false,
	}
	id, err := createRole(ctx, h.sharedDB, role, models.NewPerms())
	if err != nil {
		return nil, err
	}
	return &RoleH{
		id:       id,
		name:     roleName,
		domain:   h.domain,
		preset:   false,
		sharedDB: h.sharedDB,
	}, nil
}

func (h *RolesH) GetRoleH(ctx context.Context, roleName string) (*RoleH, error) {
	if err := h.rolesPerms.Require(models.PermManageRole); err != nil {
		return nil, err
	}
	role, err := findRoleByName(ctx, h.sharedDB, h.domain, roleName)
	if err != nil {
		return nil, err
	}
	_, err = listRolePerms(ctx, h.sharedDB, role.ID)
	if err != nil {
		return nil, err
	}

	return &RoleH{
		id:       role.ID,
		name:     role.Name,
		preset:   role.Preset,
		domain:   h.domain,
		sharedDB: h.sharedDB,
	}, nil
}

func (h *RolesH) withTx(tx DBTX) RolesH {
	return RolesH{
		contextPerms: h.contextPerms,
		rolesPerms:   h.rolesPerms,
		domain:       h.domain,
		sharedDB:     tx,
	}
}
