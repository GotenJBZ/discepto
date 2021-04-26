package routes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/utils"
)

func (routes *Routes) SubdisceptoRouter(r chi.Router) {
	r.Get("/", routes.AppHandler(routes.GetSubdisceptos))
	r.With(routes.EnforceCtx(UserHCtxKey)).Post("/", routes.AppHandler(routes.PostSubdiscepto))

	specificSub := r.With(routes.SubdiscpetoCtx)
	specificSub.Get("/{subdiscepto}", routes.AppHandler(routes.GetSubdiscepto))
	specificSub.Put("/{subdiscepto}", routes.AppHandler(routes.PutSubdiscepto))
	specificSub.Route("/{subdiscepto}/", routes.EssaysRouter)

	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Route("/{subdiscepto}/settings", routes.SubSettingsRouter)
	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Route("/{subdiscepto}/roles", routes.SubRoleRouter)
	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Route("/{subdiscepto}/members", routes.SubMembersRouter)
	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Post("/{subdiscepto}/leave", routes.AppHandler(routes.LeaveSubdiscepto))
	specificSub.With(routes.EnforceCtx(UserHCtxKey)).Post("/{subdiscepto}/join", routes.AppHandler(routes.JoinSubdiscepto))
}
func (routes *Routes) SubdiscpetoCtx(next http.Handler) http.Handler {
	return routes.AppHandler(func(w http.ResponseWriter, r *http.Request) AppError {
		userH, _ := r.Context().Value(UserHCtxKey).(*db.UserH)
		disceptoH := r.Context().Value(DiscpetoHCtxKey).(*db.DisceptoH)

		subName := chi.URLParam(r, "subdiscepto")
		subH, err := disceptoH.GetSubdisceptoH(r.Context(), subName, userH)

		if err != nil {
			return &ErrInternal{Cause: err}
		}
		ctx := context.WithValue(r.Context(), SubdisceptoHCtxKey, subH)
		next.ServeHTTP(w, r.WithContext(ctx))
		return nil
	})
}
func (routes *Routes) LeaveSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	userH := r.Context().Value(UserHCtxKey).(*db.UserH)
	subH := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)

	err := subH.RemoveMember(r.Context(), *userH)
	if err != nil {
		return &ErrInternal{Message: "Error leaving", Cause: err}
	}

	sub, err := subH.ReadView(r.Context(), userH)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	routes.tmpls.RenderHTML(w, "subdisceptoCard", sub)
	return nil
}
func (routes *Routes) JoinSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	userH := r.Context().Value(UserHCtxKey).(*db.UserH)
	subH := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)

	err := subH.AddMember(r.Context(), *userH)
	if err != nil {
		return &ErrInternal{Message: "Error joining", Cause: err}
	}
	sub, err := subH.ReadView(r.Context(), userH)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	routes.tmpls.RenderHTML(w, "subdisceptoCard", sub)
	return nil
}
func (routes *Routes) GetSubdisceptos(w http.ResponseWriter, r *http.Request) AppError {
	disceptoH, _ := r.Context().Value(DiscpetoHCtxKey).(*db.DisceptoH)
	userH, _ := r.Context().Value(UserHCtxKey).(*db.UserH)
	subs, err := routes.db.ListSubdisceptos(r.Context(), userH)
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "subdisceptos"}
	}

	data := struct {
		GlobalPerms models.GlobalPerms
		Subs        []models.SubdisceptoView
	}{
		GlobalPerms: disceptoH.Perms(),
		Subs:        subs,
	}

	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "subdisceptos"}
	}
	routes.tmpls.RenderHTML(w, "subdisceptos", data)
	return nil
}
func (routes *Routes) GetSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	userH, _ := r.Context().Value(UserHCtxKey).(*db.UserH)
	subH, _ := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)

	subData, err := subH.ReadView(r.Context(), userH)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	essays, err := subH.ListEssays(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err, Message: "Can't list essays"}
	}

	isMember := false
	var subs []string
	if userH != nil {
		var err error
		subs, err = userH.ListMySubdisceptos(r.Context())
		if err != nil {
			return &ErrInternal{Cause: err, Message: "Error getting sub membership"}
		}
		for _, s := range subs {
			if s == subH.Name() {
				isMember = true
				break
			}
		}
	}

	data := struct {
		*models.SubdisceptoView
		Essays          []models.EssayView
		IsMember        bool
		SubdisceptoList []string
		SubPerms        models.SubPerms
	}{
		SubdisceptoView: subData,
		Essays:          essays,
		IsMember:        isMember,
		SubdisceptoList: subs,
		SubPerms:        subH.Perms(),
	}
	routes.tmpls.RenderHTML(w, "subdiscepto", data)
	return nil
}
func (routes *Routes) PostSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	userH := r.Context().Value(UserHCtxKey).(*db.UserH)
	disceptoH := r.Context().Value(DiscpetoHCtxKey).(*db.DisceptoH)

	sub := models.Subdiscepto{}
	err := utils.ParseFormStruct(r, &sub)
	if err != nil {
		return &ErrBadRequest{}
	}

	_, err = disceptoH.CreateSubdiscepto(r.Context(), *userH, sub)
	if err != nil {
		return &ErrInternal{Message: "Error creating subdiscepto", Cause: err}
	}
	http.Redirect(w, r, fmt.Sprintf("/s/%s", sub.Name), http.StatusSeeOther)
	return nil
}

func (routes *Routes) PutSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	subH := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)

	sub := &models.Subdiscepto{}
	err := utils.ParseFormStruct(r, sub)
	if err != nil {
		return &ErrBadRequest{}
	}

	err = subH.Update(r.Context(), *sub)
	if err != nil {
		return &ErrInternal{Message: "Error updating subdiscepto data", Cause: err}
	}
	sub, err = subH.ReadRaw(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	routes.tmpls.RenderHTML(w, "subdisceptoForm", struct{ Subdiscepto *models.Subdiscepto }{sub})
	return nil
}
