package db

import (
	"context"
)

type RolePerms struct {
	ManageRole bool
	UpdateRole bool
	DeleteRole bool
}
type RoleH struct {
	id        int
	name      string
	domain    string
	rolePerms RolePerms
	sharedDB  DBTX
}

func (h *DisceptoH) CreateRole(ctx context.Context, roleName string) (*RoleH, error) {
	preset := false
	id, err := createRole(ctx, h.sharedDB, "discepto", roleName, preset, map[string]bool{})
	if err != nil {
		return nil, err
	}
	return &RoleH{
		id:       id,
		name:     roleName,
		domain:   "discepto",
		sharedDB: h.sharedDB,
		rolePerms: RolePerms{
			ManageRole: true,
			UpdateRole: true,
			DeleteRole: true,
		},
	}, nil
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
			ManageRole: permManageRole,
			UpdateRole: !role.Preset,
			DeleteRole: !role.Preset,
		},
	}, nil
}
func (h *SubdisceptoH) CreateRole(ctx context.Context, roleName string) (*RoleH, error) {
	preset := false
	domain := subRoleDomain(h.name)
	id, err := createRole(ctx, h.sharedDB, domain, roleName, preset, map[string]bool{})
	if err != nil {
		return nil, err
	}
	return &RoleH{
		id:       id,
		name:     roleName,
		domain:   domain,
		sharedDB: h.sharedDB,
		rolePerms: RolePerms{
			ManageRole: true,
			UpdateRole: !preset,
			DeleteRole: !preset,
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
			ManageRole: permManageRole,
			UpdateRole: !role.Preset,
			DeleteRole: !role.Preset,
		},
	}, nil
}
func (h *RoleH) ListActivePerms(ctx context.Context) (map[string]bool, error) {
	return listRolePerms(ctx, h.sharedDB, h.id)
}
func (h *RoleH) UpdatePerms(ctx context.Context, perms map[string]bool) error {
	if !h.rolePerms.UpdateRole {
		return ErrPermDenied
	}
	return setPermissions(ctx, h.sharedDB, h.id, perms)
}
func (h *RoleH) Perms() RolePerms {
	return h.rolePerms
}
func (h *RoleH) DeleteRole(ctx context.Context) error {
	if !h.rolePerms.DeleteRole {
		return ErrPermDenied
	}
	return deleteRole(ctx, h.sharedDB, h.id)
}
