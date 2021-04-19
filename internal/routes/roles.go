package routes

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/utils"
)

func (routes *Routes) GlobalRolesRouter(r chi.Router) {
	r.Use(routes.EnforceCtx(UserHCtxKey))
	r.Get("/", routes.AppHandler(routes.GetGlobalRoles))
	r.Post("/", routes.AppHandler(routes.createGlobalRole))
}
func (routes *Routes) SubRoleRouter(r chi.Router) {
	r.Use(routes.EnforceCtx(UserHCtxKey))
	r.Get("/", routes.AppHandler(routes.GetSubRoles))
	r.Post("/", routes.AppHandler(routes.createSubRole))
	r.Post("/{userID}", routes.AppHandler(routes.assignSubRole))
}
func (routes *Routes) GetGlobalRoles(w http.ResponseWriter, r *http.Request) AppError {
	routes.tmpls.RenderHTML(w, "roles", nil)
	return nil
}
func (routes *Routes) GetSubRoles(w http.ResponseWriter, r *http.Request) AppError {
	subH := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)
	roles, err := subH.ListRoles(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	data := struct {
		Roles []models.Role
	}{
		Roles: roles,
	}
	routes.tmpls.RenderHTML(w, "roles", data)
	return nil
}
func (routes *Routes) createGlobalRole(w http.ResponseWriter, r *http.Request) AppError {
	disceptoH := r.Context().Value(DiscpetoHCtxKey).(*db.DisceptoH)
	v := func(r *http.Request, formValue string) bool {
		return r.FormValue(formValue) == "true"
	}
	globalPerms := &models.GlobalPerms{}
	utils.ParsePermsForm(r, globalPerms, v)

	disceptoH.CreateGlobalRole(r.Context(), *globalPerms, r.FormValue("role_name"))
	w.Write([]byte("ok, thank you"))
	return nil
}
func (routes *Routes) assignGlobalRole(w http.ResponseWriter, r *http.Request) AppError {
	userH := r.Context().Value(UserHCtxKey).(*db.UserH)
	disceptoH := r.Context().Value(DiscpetoHCtxKey).(*db.DisceptoH)

	toUserID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}

	disceptoH.AssignGlobalRole(r.Context(), *userH, toUserID, r.FormValue("role_name"), false)
	w.Write([]byte("ok, thank you"))
	return nil
}
func (routes *Routes) createSubRole(w http.ResponseWriter, r *http.Request) AppError {
	subH := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)
	v := func(r *http.Request, formValue string) bool {
		return r.FormValue(formValue) == "true"
	}
	subPerms := &models.SubPerms{}
	utils.ParsePermsForm(r, subPerms, v)

	subH.CreateRole(r.Context(), *subPerms, r.FormValue("role_name"))
	w.Write([]byte("ok, thank you"))
	return nil
}
func (routes *Routes) assignSubRole(w http.ResponseWriter, r *http.Request) AppError {
	userH := r.Context().Value(UserHCtxKey).(*db.UserH)
	subH := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)

	toUserID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}

	subH.AssignRole(r.Context(), *userH, toUserID, r.FormValue("role_name"), false)
	w.Write([]byte("ok, thank you"))
	return nil
}
