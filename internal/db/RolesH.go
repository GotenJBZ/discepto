package db

import (
	"context"

	"gitlab.com/ranfdev/discepto/internal/models"
)

type RoleDomain string

type RolesH struct {
	contextPerms map[string]bool
	rolesPerms   struct {
		ManageRoles bool
	}
	domain   RoleDomain
	sharedDB DBTX
}

func (h *RolesH) Assign(ctx context.Context, toUser int, roleH RoleH) error {
	if !h.rolesPerms.ManageRoles ||
		!roleH.rolePerms.ManageRole ||
		roleH.domain != h.domain {
		return ErrPermDenied
	}
	newRolePerms, err := roleH.ListActivePerms(ctx)
	if err != nil {
		return err
	}
	if !isLowerRole(HigherRole(h.contextPerms), newRolePerms) {
		return ErrPermDenied
	}
	return assignRole(ctx, h.sharedDB, toUser, roleH.id)
}

func (h *RolesH) Unassign(ctx context.Context, toUser int, roleH RoleH) error {
	if !h.rolesPerms.ManageRoles || !roleH.rolePerms.ManageRole || roleH.domain != h.domain {
		return ErrPermDenied
	}
	newRolePerms, err := roleH.ListActivePerms(ctx)
	if err != nil {
		return err
	}
	if !isLowerRole(HigherRole(h.contextPerms), newRolePerms) {
		return ErrPermDenied
	}
	return unassignRole(ctx, h.sharedDB, toUser, roleH.id)
}

func (h *RolesH) ListRoles(ctx context.Context) ([]models.Role, error) {
	if !h.rolesPerms.ManageRoles {
		return nil, ErrPermDenied
	}
	return listRoles(ctx, h.sharedDB, string(h.domain))
}

func (h *RolesH) ListUserRoles(ctx context.Context, userID int) ([]models.Role, error) {
	if !h.rolesPerms.ManageRoles {
		return nil, ErrPermDenied
	}
	return listUserRoles(ctx, h.sharedDB, userID, string(h.domain))
}

func (h *RolesH) UnassignAll(ctx context.Context, userID int) error {
	return execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		roles, err := listUserRoles(ctx, tx, userID, string(h.domain))
		if err != nil {
			return err
		}
		for _, role := range roles {
			err := unassignRole(ctx, tx, userID, role.ID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (h *RolesH) CreateRole(ctx context.Context, roleName string) (*RoleH, error) {
	if !h.rolesPerms.ManageRoles {
		return nil, ErrPermDenied
	}
	preset := false
	id, err := createRole(ctx, h.sharedDB, string(h.domain), roleName, preset, map[string]bool{})
	if err != nil {
		return nil, err
	}
	return &RoleH{
		id:       id,
		name:     roleName,
		domain:   h.domain,
		sharedDB: h.sharedDB,
		rolePerms: RolePerms{
			ManageRole: true,
			UpdateRole: !preset,
			DeleteRole: !preset,
		},
	}, nil
}

func (h *RolesH) GetRoleH(ctx context.Context, roleName string) (*RoleH, error) {
	if !h.rolesPerms.ManageRoles {
		return nil, ErrPermDenied
	}
	role, err := findRoleByName(ctx, h.sharedDB, string(h.domain), roleName)
	if err != nil {
		return nil, err
	}
	perms, err := listRolePerms(ctx, h.sharedDB, role.ID)
	if err != nil {
		return nil, err
	}
	permManageRole := isLowerRole(HigherRole(h.contextPerms), perms)

	return &RoleH{
		id:       role.ID,
		name:     role.Name,
		domain:   h.domain,
		sharedDB: h.sharedDB,
		rolePerms: RolePerms{
			ManageRole: permManageRole,
			UpdateRole: !role.Preset,
			DeleteRole: !role.Preset,
		},
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
