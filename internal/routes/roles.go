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
	r.Post("/", routes.AppHandler(routes.createGlobalRole))
	r.Post("/{userID}", routes.AppHandler(routes.assignGlobalRole))
}
func (routes *Routes) SubRoleRouter(r chi.Router) {
	r.Post("/", routes.AppHandler(routes.createSubRole))
	r.Post("/{userID}", routes.AppHandler(routes.assignSubRole))
}
func (routes *Routes) createGlobalRole(w http.ResponseWriter, r *http.Request) AppError {
	user := r.Context().Value("user").(*db.UserH)
	disceptoH, err := routes.db.GetDisceptoH(r.Context(), user)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	v := func (r *http.Request, formValue string) bool {
		return r.FormValue(formValue) == "true"
	}
	globalPerms := &models.GlobalPerms{}
	utils.ParsePermsForm(r, globalPerms, v)

	disceptoH.CreateGlobalRole(r.Context(), *globalPerms, r.FormValue("role_name"))
	w.Write([]byte("ok, thank you"))
	return nil
}
func (routes *Routes) assignGlobalRole(w http.ResponseWriter, r *http.Request) AppError {
	user := r.Context().Value("user").(*db.UserH)
	disceptoH, err := routes.db.GetDisceptoH(r.Context(), user)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	toUserID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}

	disceptoH.AssignGlobalRole(r.Context(), *user, toUserID, r.FormValue("role_name"), false)
	w.Write([]byte("ok, thank you"))
	return nil
}
func (routes *Routes) createSubRole(w http.ResponseWriter, r *http.Request) AppError {
	user := r.Context().Value("user").(*db.UserH)
	subH, err := routes.db.GetSubdisceptoH(r.Context(), chi.URLParam(r, "subdiscepto"), user)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	v := func (r *http.Request, formValue string) bool {
		return r.FormValue(formValue) == "true"
	}
	subPerms := &models.SubPerms{}
	utils.ParsePermsForm(r, subPerms, v)

	subH.CreateRole(r.Context(), *subPerms, r.FormValue("role_name"))
	w.Write([]byte("ok, thank you"))
	return nil
}
func (routes *Routes) assignSubRole(w http.ResponseWriter, r *http.Request) AppError {
	user := r.Context().Value("user").(*db.UserH)
	subH, err := routes.db.GetSubdisceptoH(r.Context(), chi.URLParam(r, "subdiscepto"), user)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	toUserID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}

	subH.AssignRole(r.Context(), *user, toUserID, r.FormValue("role_name"), false)
	w.Write([]byte("ok, thank you"))
	return nil
}
