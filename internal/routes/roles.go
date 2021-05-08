package routes

import (
	"context"
	"errors"
	"net/http"
	"path"
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
	r.Get("/{roleName}", routes.AppHandler(routes.getRolePerms))
	r.Put("/{roleName}", routes.AppHandler(routes.putRolePerms))
	r.Delete("/{roleName}", routes.AppHandler(routes.deleteRole))
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
	AssignRole(ctx context.Context, byUser db.UserH, toUser int, roleH db.RoleH) error
	UnassignRole(ctx context.Context, toUser int, roleH db.RoleH) error
	ListMembers(ctx context.Context) ([]models.Member, error)
	ListRoles(ctx context.Context) ([]models.Role, error)
	ListAvailablePerms() map[string]bool
	GetRoleH(ctx context.Context, roleName string) (*db.RoleH, error)
}

func (routes *Routes) assignRole(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	roleManager := GetRoleManager(r)
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}
	roleH, err := roleManager.GetRoleH(r.Context(), r.FormValue("roleName"))
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	err = roleManager.AssignRole(r.Context(), *userH, userID, *roleH)
	pgErr := &pgconn.PgError{}
	if err != nil && !(errors.As(err, &pgErr) && pgErr.Code == "23505") {
		return &ErrInternal{Cause: err}
	}
	return routes.renderMembers(w, r)
}

func (routes *Routes) unassignRole(w http.ResponseWriter, r *http.Request) AppError {
	roleManager := GetRoleManager(r)
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}
	roleH, err := roleManager.GetRoleH(r.Context(), chi.URLParam(r, "roleName"))
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	err = roleManager.UnassignRole(r.Context(), userID, *roleH)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	return routes.renderMembers(w, r)
}

func (routes *Routes) getRolePerms(w http.ResponseWriter, r *http.Request) AppError {
	roleManager := GetRoleManager(r)
	roleName := chi.URLParam(r, "roleName")
	roleH, err := roleManager.GetRoleH(r.Context(), roleName)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	activePerms, err := roleH.ListActivePerms(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	availablePerms := roleManager.ListAvailablePerms()
	routes.tmpls.RenderHTML(w, "permissions", struct {
		RoleName       string
		AvailablePerms map[string]bool
		ActivePerms    map[string]bool
	}{
		RoleName:       roleName,
		AvailablePerms: availablePerms,
		ActivePerms:    activePerms,
	})
	return nil
}
func (routes *Routes) putRolePerms(w http.ResponseWriter, r *http.Request) AppError {
	roleManager := GetRoleManager(r)

	r.ParseForm()
	perms := map[string]bool{}
	for k, v := range r.Form {
		if v[0] == "on" {
			perms[k] = true
		}
	}
	roleH, err := roleManager.GetRoleH(r.Context(), chi.URLParam(r, "roleName"))
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	err = roleH.UpdatePerms(r.Context(), perms)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	return routes.getRolePerms(w, r)
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
func (routes *Routes) deleteRole(w http.ResponseWriter, r *http.Request) AppError {
	roleManager := GetRoleManager(r)
	roleName := chi.URLParam(r, "roleName")
	roleH, err := roleManager.GetRoleH(r.Context(), roleName)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	err = roleH.DeleteRole(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	path := path.Dir(r.URL.Path)
	w.Header().Add("HX-Redirect", path)
	http.Redirect(w, r, path, http.StatusAccepted)
	return nil
}
