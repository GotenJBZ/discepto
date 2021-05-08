package db

import (
	"context"
)

type RolePerms struct {
	ManageRole bool
}
type RoleH struct {
	id        int
	name      string
	domain    string
	rolePerms RolePerms
	sharedDB  DBTX
}

func (h *DisceptoH) GetRoleH(ctx context.Context, roleName string) (*RoleH, error) {
	if !h.globalPerms.ManageGlobalRole {
		return nil, ErrPermDenied
	}
	role, err := findRoleByName(ctx, h.sharedDB, "discepto", roleName)
	if err != nil {
		return nil, err
	}
	perms, err := listRolePerms(ctx, h.sharedDB, role.ID)
	if err != nil {
		return nil, err
	}
	permManageRole := isLowerRole(HigherRole(h.globalPerms.ToBoolMap()), perms)

	return &RoleH{
		id:       role.ID,
		name:     role.Name,
		domain:   role.Domain,
		sharedDB: h.sharedDB,
		rolePerms: RolePerms{
			ManageRole: !role.Preset && permManageRole,
		},
	}, nil
}
func (h *SubdisceptoH) GetRoleH(ctx context.Context, roleName string) (*RoleH, error) {
	if !h.subPerms.ManageRole {
		return nil, ErrPermDenied
	}
	role, err := findRoleByName(ctx, h.sharedDB, subRoleDomain(h.name), roleName)
	if err != nil {
		return nil, err
	}
	perms, err := listRolePerms(ctx, h.sharedDB, role.ID)
	if err != nil {
		return nil, err
	}
	permManageRole := isLowerRole(HigherRole(h.subPerms.ToBoolMap()), perms)

	return &RoleH{
		id:       role.ID,
		name:     role.Name,
		domain:   role.Domain,
		sharedDB: h.sharedDB,
		rolePerms: RolePerms{
			ManageRole: !role.Preset && permManageRole,
		},
	}, nil
}
func (h *RoleH) ListActivePerms(ctx context.Context) (map[string]bool, error) {
	return listRolePerms(ctx, h.sharedDB, h.id)
}
func (h *RoleH) UpdatePerms(ctx context.Context, perms map[string]bool) error {
	if !h.rolePerms.ManageRole {
		return ErrPermDenied
	}
	return setPermissions(ctx, h.sharedDB, h.id, perms)
}
func (h *RoleH) DeleteRole(ctx context.Context) error {
	if !h.rolePerms.ManageRole {
		return ErrPermDenied
	}
	return deleteRole(ctx, h.sharedDB, h.id)
}
