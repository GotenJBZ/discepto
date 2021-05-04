package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/jackc/pgconn"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func (routes *Routes) GlobalRolesRouter(r chi.Router) {
	r.Use(RoleManagerCtx(GetRoleManagerDiscepto))
	routes.roleRouter(r)
}
func (routes *Routes) SubRoleRouter(r chi.Router) {
	r.Use(RoleManagerCtx(GetRoleManagerSubdiscepto))
	routes.roleRouter(r)
}
func (routes *Routes) roleRouter(r chi.Router) {
	r.Use(routes.EnforceCtx(UserHCtxKey))
	r.Get("/", routes.AppHandler(routes.listRoles))
}

func GetRoleManagerDiscepto(r *http.Request) RoleManager {
	return GetDisceptoH(r)
}
func GetRoleManagerSubdiscepto(r *http.Request) RoleManager {
	return GetSubdisceptoH(r)
}

type RoleManagerExtract = func(r *http.Request) RoleManager

func RoleManagerCtx(extract RoleManagerExtract) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleManager := extract(r)
			ctx := context.WithValue(r.Context(), RoleManagerKey, roleManager)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type RoleManager interface {
	AssignRole(ctx context.Context, userH db.UserH, userID int, roleID int) error
	UnassignRole(ctx context.Context, userID int, roleID int) error
	ListMembers(ctx context.Context) ([]models.Member, error)
	ListRoles(ctx context.Context) ([]models.Role, error)
}

func (routes *Routes) assignRole(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	roleManager := GetRoleManager(r)
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}
	roleID, err := strconv.Atoi(r.FormValue("role"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}
	err = roleManager.AssignRole(r.Context(), *userH, userID, roleID)
	pgErr := &pgconn.PgError{}
	if err != nil && !(errors.As(err, &pgErr) && pgErr.Code == "23505") {
		return &ErrInternal{Cause: err}
	}
	return routes.renderMembers(w, r)
}

func (routes *Routes) unassignRole(w http.ResponseWriter, r *http.Request) AppError {
	roleManager := GetRoleManager(r)
	fmt.Println(roleManager)
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}
	roleID, err := strconv.Atoi(chi.URLParam(r, "roleID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}
	err = roleManager.UnassignRole(r.Context(), userID, roleID)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	return routes.renderMembers(w, r)
}

// Should use better number
const RoleManagerKey = disceptoCtxKey(100)

func GetRoleManager(r *http.Request) RoleManager {
	return r.Context().Value(RoleManagerKey).(RoleManager)
}
func (routes *Routes) listRoles(w http.ResponseWriter, r *http.Request) AppError {
	roleManager := GetRoleManager(r)
	roles, err := roleManager.ListRoles(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	data := struct {
		Subdiscepto *models.SubdisceptoView
		Roles       []models.Role
	}{
		Subdiscepto: nil,
		Roles:       roles,
	}
	routes.tmpls.RenderHTML(w, "roles", data)
	return nil
}
