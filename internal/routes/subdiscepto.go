package routes

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func (routes *Routes) SubdisceptoRouter(r chi.Router) {
	r.Post("/{name}/leave", routes.AppHandler(routes.LeaveSubdiscepto))
	r.Post("/{name}/join", routes.AppHandler(routes.JoinSubdiscepto))
	r.Get("/{name}/{id}", routes.AppHandler(routes.GetEssay))
	r.Get("/{name}", routes.AppHandler(routes.GetSubdiscepto))
	r.Get("/", routes.AppHandler(routes.GetSubdisceptos))
	r.Post("/", routes.AppHandler(routes.PostSubdiscepto))
}
func (routes *Routes) LeaveSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &ErrMustLogin{}
	}
	err := routes.db.LeaveSubdiscepto(chi.URLParam(r, "name"), user.ID)
	if err != nil {
		return &ErrInternal{Message: "Error leaving", Cause: err}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func (routes *Routes) JoinSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &ErrMustLogin{}
	}
	err := routes.db.JoinSubdiscepto(chi.URLParam(r, "name"), user.ID)
	if err != nil {
		return &ErrInternal{Message: "Error joining", Cause: err}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func (routes *Routes) GetSubdisceptos(w http.ResponseWriter, r *http.Request) AppError {
	subs, err := routes.db.ListSubdisceptos()
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "subdisceptos"}
	}
	routes.tmpls.RenderHTML(w, "subdisceptos", subs)
	return nil
}
func (routes *Routes) GetSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	name := chi.URLParam(r, "name")
	sub, err := routes.db.GetSubdiscepto(name)
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "subdiscepto"}
	}
	essays, err := routes.db.ListEssays(name)
	if err != nil {
		return &ErrInternal{Cause: err, Message: "Can't list essays"}
	}

	isMember := false
	user, ok := r.Context().Value("user").(*models.User)
	if ok {
		subs, err := routes.db.ListMySubdisceptos(user.ID)
		if err != nil {
			return &ErrInternal{Cause: err, Message: "Error getting sub membership"}
		}
		for _, s := range subs {
			if s == name {
				isMember = true
				break
			}
		}
	}
	data := struct {
		Name        string
		Description string
		Essays      []*models.Essay
		IsMember    bool
	}{
		Name:        sub.Name,
		Description: sub.Description,
		Essays:      essays,
		IsMember:    isMember,
	}
	routes.tmpls.RenderHTML(w, "subdiscepto", data)
	return nil
}
func (routes *Routes) PostSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &ErrMustLogin{}
	}

	sub := &models.Subdiscepto{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}
	err := routes.db.CreateSubdiscepto(sub, user.ID)
	if err != nil {
		return &ErrInternal{Message: "Error creating subdiscepto", Cause: err}
	}
	http.Redirect(w, r, fmt.Sprintf("/s/%s", sub.Name), http.StatusSeeOther)
	return nil
}
func (routes *Routes) GetEssay(w http.ResponseWriter, r *http.Request) AppError {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}
	essay, err := routes.db.GetEssay(id)
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	essay.Upvotes, essay.Downvotes, err = routes.db.CountVotes(essay.ID)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	routes.tmpls.RenderHTML(w, "essay", essay)
	return nil
}
