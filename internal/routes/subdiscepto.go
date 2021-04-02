package routes

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func (routes *Routes) SubdisceptoRouter(r chi.Router) {
	r.Post("/{subdiscepto}/leave", routes.AppHandler(routes.LeaveSubdiscepto))
	r.Post("/{subdiscepto}/join", routes.AppHandler(routes.JoinSubdiscepto))
	r.Route("/{subdiscepto}/", routes.EssaysRouter)
	r.Get("/{subdiscepto}", routes.AppHandler(routes.GetSubdiscepto))
	r.Route("/roles", routes.SubRoleRouter)
	r.Get("/", routes.AppHandler(routes.GetSubdisceptos))
	r.Post("/", routes.AppHandler(routes.PostSubdiscepto))
}
func (routes *Routes) LeaveSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value(UserHCtxKey).(*db.UserH)
	if !ok {
		return &ErrMustLogin{}
	}
	subdiscepto := chi.URLParam(r, "subdiscepto")
	subH, err := routes.db.GetSubdisceptoH(r.Context(), subdiscepto, user)
	if err != nil {
		return &ErrNotFound{Cause: err}
	}

	err = subH.RemoveMember(r.Context(), *user)
	if err != nil {
		return &ErrInternal{Message: "Error leaving", Cause: err}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func (routes *Routes) JoinSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value(UserHCtxKey).(*db.UserH)
	if !ok {
		return &ErrMustLogin{}
	}
	subdiscepto := chi.URLParam(r, "subdiscepto")
	subH, err := routes.db.GetSubdisceptoH(r.Context(), subdiscepto, user)
	if err != nil {
		return &ErrNotFound{Cause: err}
	}

	err = subH.AddMember(r.Context(), *user)
	if err != nil {
		return &ErrInternal{Message: "Error joining", Cause: err}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func (routes *Routes) GetSubdisceptos(w http.ResponseWriter, r *http.Request) AppError {
	subs, err := routes.db.ListSubdisceptos(r.Context())
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "subdisceptos"}
	}
	routes.tmpls.RenderHTML(w, "subdisceptos", subs)
	return nil
}
func (routes *Routes) GetSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	subdiscepto := chi.URLParam(r, "subdiscepto")
	user, ok := r.Context().Value(UserHCtxKey).(*db.UserH)
	subH, err := routes.db.GetSubdisceptoH(r.Context(), subdiscepto, user)
	if err != nil {
		return &ErrNotFound{Cause: err}
	}

	subData, err := subH.Read(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	essays, err := subH.ListEssays(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err, Message: "Can't list essays"}
	}

	isMember := false
	var subs []string
	if ok {
		var err error
		subs, err = user.ListMySubdisceptos(r.Context())
		if err != nil {
			return &ErrInternal{Cause: err, Message: "Error getting sub membership"}
		}
		for _, s := range subs {
			if s == subdiscepto {
				isMember = true
				break
			}
		}
	}
	data := struct {
		Name            string
		Description     string
		Essays          []*models.Essay
		IsMember        bool
		SubdisceptoList []string
	}{
		Name:            subData.Name,
		Description:     subData.Description,
		Essays:          essays,
		IsMember:        isMember,
		SubdisceptoList: subs,
	}
	routes.tmpls.RenderHTML(w, "subdiscepto", data)
	return nil
}
func (routes *Routes) PostSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value(UserHCtxKey).(*db.UserH)
	if !ok {
		return &ErrMustLogin{}
	}

	sub := &models.Subdiscepto{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Public:      r.FormValue("privacy") == "public", // TODO: Use checkbox instead of radio in html
	}

	disceptoH, err := routes.db.GetDisceptoH(r.Context(), user)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	_, err = disceptoH.CreateSubdiscepto(r.Context(), *user, sub)
	if err != nil {
		return &ErrInternal{Message: "Error creating subdiscepto", Cause: err}
	}
	http.Redirect(w, r, fmt.Sprintf("/s/%s", sub.Name), http.StatusSeeOther)
	return nil
}
