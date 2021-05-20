package domain

import (
	"context"
)

type RBACRepo interface {
	ListRoles(ctx context.Context, domain string) ([]Role, error)
	ListRolePerms(ctx context.Context, domain string, name string) (map[string]bool, error)
	ListUserRoles(ctx context.Context, domain string, userID int) ([]Role, error)
	FindRoleByName(ctx context.Context, domain string, name string) (*Role, error)
	ListUserPerms(ctx context.Context, domain string, userID int) (map[string]bool, error)
	AssignRole(ctx context.Context, userID int, roleID int) error
	UnassignRole(ctx context.Context, userID int, roleID int) error
  UnassignAll(ctx context.Context, userID int) error
	CreateRole(ctx context.Context, role Role, m map[string]bool) (*int, error)
	SetPermissions(ctx context.Context, roleID int, m map[string]bool) error
	DeleteRole(ctx context.Context, roleID int) error
}

type RBACService interface {
	ListRoles(ctx context.Context) ([]Role, error)
	ListRolePerms(ctx context.Context, name string) (map[string]bool, error)
	ListUserRoles(ctx context.Context, userID int) ([]Role, error)
	ListUserPerms(ctx context.Context, userID int) (map[string]bool, error)
	AssignRole(ctx context.Context, userID int, role Role) error
	UnassignRole(ctx context.Context, userID int, role Role) error
	CreateRole(ctx context.Context, role Role, m map[string]bool) (*int, error)
	SetPermissions(ctx context.Context, roleID int, m map[string]bool) error
	DeleteRole(ctx context.Context, roleID int) error
}

type DisceptoRepo interface {
	func GetDisceptoH(ctx context.Context, uH *UserH, rbacRepo domain.RBACRepo) (*DisceptoH, error)
	func Perms() domain.GlobalPerms
	func ListMembers(ctx context.Context) ([]domain.Member, error)
	func ReadPublicUser(ctx context.Context, userID int) (*domain.UserView, error)
	func CreateSubdiscepto(ctx context.Context, uH UserH, subd domain.Subdiscepto) (*SubdisceptoH, error)
	func ListAvailablePerms() map[string]bool
	func createSubdiscepto(ctx context.Context, uH UserH, subd domain.Subdiscepto) (*SubdisceptoH, error)
		err := execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error
	func DeleteReport(ctx context.Context, report *domain.Report) error
	func ListRecentEssaysIn(ctx context.Context, subs []string) ([]domain.EssayView, error)
	func ListSubdisceptos(ctx context.Context, userH *UserH) ([]domain.SubdisceptoView, error)
	func SearchByTags(ctx context.Context, tags []string) ([]domain.EssayView, error)
	func scanEssays(ctx context.Context, rows pgx.Rows, tags []string) ([]domain.EssayView, error)
	func SearchByThesis(ctx context.Context, title string) ([]domain.EssayView, error)
	func ListUserEssays(ctx context.Context, userID int) ([]domain.EssayView, error)
	func readPublicUser(ctx context.Context, db DBTX, userID int) (*domain.UserView, error)
}
