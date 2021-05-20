package domain

import "context"

const RoleDefault = 0
const RoleAdmin = -123
const SubRoleAdminPreset = -123

type HigherPerms map[string]bool

type RoleDomain string
type Role struct {
	ID     int
	Name   string
	Preset bool
	Domain string
}

func areLowerPerms(oldPerms HigherPerms, newPerms map[string]bool) bool {
	for k := range map[string]bool(newPerms) {
		if v, ok := oldPerms[k]; !ok || !v {
			return false
		}
	}
	return true
}

type rbacServicePerms struct {
	ManageRoles bool
}
type rbacService struct {
	contextPerms map[string]bool
	rolesPerms   rbacServicePerms
	domain       RoleDomain
	repo         RBACRepo
}

func NewRBACService(repo RBACRepo, contextPerms map[string]bool, manageRole bool, domain string) RBACService {
	return &rbacService{
		contextPerms,
		rbacServicePerms{manageRole},
		RoleDomain(domain),
		repo,
	}
}

func (h *rbacService) AssignRole(ctx context.Context, toUser int, role Role) error {
	if !h.rolesPerms.ManageRoles {
		return ErrPermDenied
	}
	newRolePerms, err := h.repo.ListRolePerms(ctx, string(h.domain), role.Name)
	if err != nil {
		return err
	}
	if !areLowerPerms(HigherPerms(h.contextPerms), newRolePerms) {
		return ErrPermDenied
	}
	return h.repo.AssignRole(ctx, toUser, role.ID)
}

func (h *rbacService) UnassignRole(ctx context.Context, toUser int, role Role) error {
	if !h.rolesPerms.ManageRoles {
		return ErrPermDenied
	}
	newRolePerms, err := h.repo.ListRolePerms(ctx, string(h.domain), role.Name)
	if err != nil {
		return err
	}
	if !areLowerPerms(HigherPerms(h.contextPerms), newRolePerms) {
		return ErrPermDenied
	}
	return h.repo.AssignRole(ctx, toUser, role.ID)
}

func (h *rbacService) ListRoles(ctx context.Context) ([]Role, error) {
	if !h.rolesPerms.ManageRoles {
		return nil, ErrPermDenied
	}
	return h.repo.ListRoles(ctx, string(h.domain))
}
func (h *rbacService) ListRolePerms(ctx context.Context, roleName string) (map[string]bool, error) {
	if !h.rolesPerms.ManageRoles {
		return nil, ErrPermDenied
	}
	return h.repo.ListRolePerms(ctx, string(h.domain), roleName)
}

func (h *rbacService) ListUserPerms(ctx context.Context, userID int) (map[string]bool, error) {
	if !h.rolesPerms.ManageRoles {
		return nil, ErrPermDenied
	}
	return h.repo.ListUserPerms(ctx, string(h.domain), userID)
}

func (h *rbacService) ListUserRoles(ctx context.Context, userID int) ([]Role, error) {
	if !h.rolesPerms.ManageRoles {
		return nil, ErrPermDenied
	}
	return h.repo.ListUserRoles(ctx, string(h.domain), userID)
}

func (h *rbacService) UnassignAll(ctx context.Context, userID int) error {
	return h.repo.UnassignAll(ctx, userID)
}

func (h *rbacService) CreateRole(ctx context.Context, role Role, perms map[string]bool) (*int, error) {
	if !h.rolesPerms.ManageRoles {
		return nil, ErrPermDenied
	}
	id, err := h.repo.CreateRole(ctx, role, map[string]bool{})
	if err != nil {
		return nil, err
	}
	return id, nil
}
func (h *rbacService) DeleteRole(ctx context.Context, roleID int) error {
	if !h.rolesPerms.ManageRoles {
		return ErrPermDenied
	}
	return h.repo.DeleteRole(ctx, roleID)
}
func (h *rbacService) SetPermissions(ctx context.Context, roleID int, m map[string]bool) error {
	if !h.rolesPerms.ManageRoles {
		return ErrPermDenied
	}
	return h.repo.SetPermissions(ctx, roleID, m)
}
