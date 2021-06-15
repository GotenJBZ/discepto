package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/go-chi/chi/v5"
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
	r.Get("/", routes.listRoles)
	r.Get("/{roleName}", routes.getRolePerms)
	r.Post("/", routes.postNewRole)
	r.Put("/{roleName}", routes.putRolePerms)
	r.Delete("/{roleName}", routes.deleteRole)
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
	Assign(ctx context.Context, toUser int, roleH db.RoleH) error
	Unassign(ctx context.Context, toUser int, roleH db.RoleH) error
	ListMembers(ctx context.Context) ([]models.Member, error)
	ListRoles(ctx context.Context) ([]models.Role, error)
	ListAvailablePerms() models.Perms
	GetRoleH(ctx context.Context, roleName string) (*db.RoleH, error)
	CreateRole(ctx context.Context, roleName string) (*db.RoleH, error)
}

func (routes *Routes) assignRole(w http.ResponseWriter, r *http.Request) {
	roleManager := GetRoleManager(r)
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	roleH, err := roleManager.GetRoleH(r.Context(), r.FormValue("roleName"))
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	err = roleManager.Assign(r.Context(), userID, *roleH)
	pgErr := &pgconn.PgError{}
	if err != nil && !(errors.As(err, &pgErr) && pgErr.Code == "23505") {
		routes.HandleErr(w, r, err)
		return
	}
	routes.renderMembers(w, r)
}

func (routes *Routes) unassignRole(w http.ResponseWriter, r *http.Request) {
	roleManager := GetRoleManager(r)
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	roleH, err := roleManager.GetRoleH(r.Context(), chi.URLParam(r, "roleName"))
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	err = roleManager.Unassign(r.Context(), userID, *roleH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	routes.renderMembers(w, r)
}

func (routes *Routes) getRolePerms(w http.ResponseWriter, r *http.Request) {
	roleManager := GetRoleManager(r)
	roleName := chi.URLParam(r, "roleName")
	roleH, err := roleManager.GetRoleH(r.Context(), roleName)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	activePerms, err := roleH.ListActivePerms(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	availablePerms := roleManager.ListAvailablePerms()
	routes.tmpls.RenderHTML(w, "permissions", struct {
		RoleName       string
		AvailablePerms models.Perms
		ActivePerms    models.Perms
		RoleH          *db.RoleH
	}{
		RoleName:       roleName,
		AvailablePerms: availablePerms,
		ActivePerms:    activePerms,
		RoleH:          roleH,
	})
}
func (routes *Routes) putRolePerms(w http.ResponseWriter, r *http.Request) {
	roleManager := GetRoleManager(r)

	r.ParseForm()
	var perms models.Perms
	{
		tmpPerms := []models.Perm{}
		for k, v := range r.Form {
			if v[0] == "on" {
				tmpPerms = append(tmpPerms, models.Perm(k))
			}
		}
		perms = models.NewPerms(tmpPerms...)
	}
	roleH, err := roleManager.GetRoleH(r.Context(), chi.URLParam(r, "roleName"))
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	err = roleH.UpdatePerms(r.Context(), perms)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	routes.getRolePerms(w, r)
}

// Should use better number
const RoleManagerKey = disceptoCtxKey(100)

func GetRoleManager(r *http.Request) RoleManager {
	return r.Context().Value(RoleManagerKey).(RoleManager)
}
func (routes *Routes) listRoles(w http.ResponseWriter, r *http.Request) {
	roleManager := GetRoleManager(r)
	roles, err := roleManager.ListRoles(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	data := struct {
		Subdiscepto *models.SubdisceptoView
		Roles       []models.Role
	}{
		Subdiscepto: nil,
		Roles:       roles,
	}
	routes.tmpls.RenderHTML(w, "roles", data)
}
func (routes *Routes) deleteRole(w http.ResponseWriter, r *http.Request) {
	roleManager := GetRoleManager(r)
	roleName := chi.URLParam(r, "roleName")
	roleH, err := roleManager.GetRoleH(r.Context(), roleName)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	err = roleH.DeleteRole(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	path := path.Dir(r.URL.Path)
	w.Header().Add("HX-Redirect", path)
	http.Redirect(w, r, path, http.StatusAccepted)
}
func (routes *Routes) postNewRole(w http.ResponseWriter, r *http.Request) {
	roleManager := GetRoleManager(r)
	roleName := r.FormValue("roleName")
	if roleName == "" {
		err := &ErrBadRequest{Cause: errors.New("fill required inputs")}
		routes.HandleErr(w, r, err)
		return
	}
	_, err := roleManager.CreateRole(r.Context(), roleName)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	path := path.Join(r.URL.Path, roleName)
	fmt.Println(path)
	w.Header().Add("HX-Redirect", path)
	http.Redirect(w, r, path, http.StatusSeeOther)
}
