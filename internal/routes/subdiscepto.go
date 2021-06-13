package routes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/utils"
)

type SubdisceptoPageData struct {
	*models.SubdisceptoView
	Essays          []models.EssayView
	SubdisceptoList []models.SubdisceptoView
	SubPerms        models.Perms
}

func (routes *Routes) SubdisceptoRouter(r chi.Router) {
	r.Get("/", routes.GetSubdisceptos)
	r.With(routes.EnforceCtx(UserHCtxKey)).Post("/", routes.PostSubdiscepto)

	specificSub := r.With(routes.SubdiscpetoCtx)
	specificSub.Get("/{subdiscepto}", routes.GetSubdiscepto)
	specificSub.Put("/{subdiscepto}", routes.PutSubdiscepto)
	specificSub.Route("/{subdiscepto}/", routes.EssaysRouter)

	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Route("/{subdiscepto}/settings", routes.SubSettingsRouter)
	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Route("/{subdiscepto}/roles", routes.SubRoleRouter)
	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Route("/{subdiscepto}/members", routes.SubMembersRouter)
	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Post("/{subdiscepto}/leave", routes.LeaveSubdiscepto)
	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Post("/{subdiscepto}/join", routes.JoinSubdiscepto)
	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Route("/{subdiscepto}/reports", routes.SubReportsRouter)
}
func (routes *Routes) SubdiscpetoCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userH := GetUserH(r)
		disceptoH := GetDisceptoH(r)

		subName := chi.URLParam(r, "subdiscepto")
		subH, err := disceptoH.GetSubdisceptoH(r.Context(), subName, userH)

		if err != nil {
			routes.HandleErr(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), SubdisceptoHCtxKey, subH)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	})
}
func (routes *Routes) LeaveSubdiscepto(w http.ResponseWriter, r *http.Request) {
	userH := GetUserH(r)
	disceptoH := GetDisceptoH(r)
	subH := GetSubdisceptoH(r)

	err := subH.RemoveMember(r.Context(), *userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	sub, err := subH.ReadView(r.Context(), userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	mySubs, err := disceptoH.ListUserSubdisceptos(r.Context(), userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	subName := chi.URLParam(r, "subdiscepto")
	subH, err = disceptoH.GetSubdisceptoH(r.Context(), subName, userH)
	if err != nil {
		routes.HandleErr(w, r, err)
	}
	routes.tmpls.RenderHTML(w, "subdiscepto", SubdisceptoPageData{
		SubdisceptoView: sub,
		Essays:          []models.EssayView{},
		SubdisceptoList: mySubs,
		SubPerms:        subH.Perms(),
	})
	return
}
func (routes *Routes) JoinSubdiscepto(w http.ResponseWriter, r *http.Request) {
	userH := GetUserH(r)
	disceptoH := GetDisceptoH(r)
	subH := GetSubdisceptoH(r)

	err := subH.AddMember(r.Context(), *userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	sub, err := subH.ReadView(r.Context(), userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	mySubs, err := disceptoH.ListUserSubdisceptos(r.Context(), userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	subName := chi.URLParam(r, "subdiscepto")
	subH, err = disceptoH.GetSubdisceptoH(r.Context(), subName, userH)
	if err != nil {
		routes.HandleErr(w, r, err)
	}
	routes.tmpls.RenderHTML(w, "subdiscepto", SubdisceptoPageData{
		SubdisceptoView: sub,
		Essays:          []models.EssayView{},
		SubdisceptoList: mySubs,
		SubPerms:        subH.Perms(),
	})
	return
}
func (routes *Routes) GetSubdisceptos(w http.ResponseWriter, r *http.Request) {
	disceptoH := GetDisceptoH(r)
	userH := GetUserH(r)
	subs, err := routes.db.ListSubdisceptos(r.Context(), userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	data := struct {
		GlobalPerms models.Perms
		Subs        []models.SubdisceptoView
	}{
		GlobalPerms: disceptoH.Perms(),
		Subs:        subs,
	}

	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	routes.tmpls.RenderHTML(w, "subdisceptos", data)
	return
}
func (routes *Routes) GetSubdiscepto(w http.ResponseWriter, r *http.Request) {
	userH := GetUserH(r)
	disceptoH := GetDisceptoH(r)
	subH := GetSubdisceptoH(r)

	subData, err := subH.ReadView(r.Context(), userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	essays, err := subH.ListEssays(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	mySubs, err := disceptoH.ListUserSubdisceptos(r.Context(), userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	routes.tmpls.RenderHTML(w, "subdiscepto", SubdisceptoPageData{
		SubdisceptoView: subData,
		Essays:          essays,
		SubdisceptoList: mySubs,
		SubPerms:        subH.Perms(),
	})
	return
}
func (routes *Routes) PostSubdiscepto(w http.ResponseWriter, r *http.Request) {
	userH := GetUserH(r)
	disceptoH := GetDisceptoH(r)

	subReq := models.SubdisceptoReq{}
	err := utils.ParseFormStruct(r, &subReq)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	_, err = disceptoH.CreateSubdiscepto(r.Context(), *userH, &subReq)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/s/%s", subReq.Name), http.StatusSeeOther)
	return
}

func (routes *Routes) PutSubdiscepto(w http.ResponseWriter, r *http.Request) {
	subH := GetSubdisceptoH(r)

	subReq := &models.SubdisceptoReq{}
	err := utils.ParseFormStruct(r, subReq)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	subReq.Name = subH.Name()

	err = subH.Update(r.Context(), subReq)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	routes.tmpls.RenderHTML(w, "subdisceptoForm", struct{ Subdiscepto *models.SubdisceptoReq }{subReq})
	return
}
