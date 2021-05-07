package db

import (
	"context"
	"fmt"
)

type RolePerms struct {
	ManageRole bool
}
type RoleH struct {
	id              int
	name            string
	domain          string
	rolePerms       RolePerms
	changeablePerms map[string]bool
	sharedDB        DBTX
}

func (h *DisceptoH) GetRoleH(ctx context.Context, roleName string) (*RoleH, error) {
	if !h.globalPerms.ManageGlobalRole {
		return nil, ErrPermDenied
	}
	role, err := findRoleByName(ctx, h.sharedDB, "discepto", roleName)
	if err != nil {
		return nil, err
	}
	return &RoleH{
		id:              role.ID,
		name:            role.Name,
		domain:          role.Domain,
		sharedDB:        h.sharedDB,
		changeablePerms: h.globalPerms.ToBoolMap(),
		rolePerms: RolePerms{
			ManageRole: !role.Preset,
		},
	}, nil
}
func (h *SubdisceptoH) GetRoleH(ctx context.Context, roleName string) (*RoleH, error) {
	if !h.subPerms.ManageRole {
		return nil, ErrPermDenied
	}
	role, err := findRoleByName(ctx, h.sharedDB, subRoleDomain(h.name), roleName)
	fmt.Println(err, roleName)
	if err != nil {
		return nil, err
	}
	return &RoleH{
		id:              role.ID,
		name:            role.Name,
		domain:          role.Domain,
		sharedDB:        h.sharedDB,
		changeablePerms: h.subPerms.ToBoolMap(),
		rolePerms: RolePerms{
			ManageRole: !role.Preset,
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
	for k, v := range perms {
		if !v {
			delete(perms, k)
			continue
		}
		if _, ok := h.changeablePerms[k]; !ok {
			return ErrPermDenied
		}
	}
	return setPermissions(ctx, h.sharedDB, h.id, perms)
}
