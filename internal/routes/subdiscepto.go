package routes

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/server"
)

func SubdisceptoRouter(r chi.Router) {
	r.Post("/{name}/leave", AppHandler(LeaveSubdiscepto))
	r.Post("/{name}/join", AppHandler(JoinSubdiscepto))
	r.Get("/{name}/{id}", AppHandler(GetEssay))
	r.Get("/{name}", AppHandler(GetSubdiscepto))
	r.Get("/", AppHandler(GetSubdisceptos))
	r.Post("/", AppHandler(PostSubdiscepto))
}
func LeaveSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &ErrMustLogin{}
	}
	err := db.LeaveSubdiscepto(chi.URLParam(r, "name"), user.ID)
	if err != nil {
		return &ErrInternal{Message: "Error leaving", Cause: err}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func JoinSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &ErrMustLogin{}
	}
	err := db.JoinSubdiscepto(chi.URLParam(r, "name"), user.ID)
	if err != nil {
		return &ErrInternal{Message: "Error joining", Cause: err}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func GetSubdisceptos(w http.ResponseWriter, r *http.Request) AppError {
	subs, err := db.ListSubdisceptos()
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "subdisceptos"}
	}
	server.RenderHTML(w, "subdisceptos", subs)
	return nil
}
func GetSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	name := chi.URLParam(r, "name")
	sub, err := db.GetSubdiscepto(name)
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "subdiscepto"}
	}
	essays, err := db.ListEssays(name)
	if err != nil {
		return &ErrInternal{Cause: err, Message: "Can't list essays"}
	}

	isMember := false
	user, ok := r.Context().Value("user").(*models.User)
	if ok {
		subs, err := db.ListMySubdisceptos(user.ID)
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
	server.RenderHTML(w, "subdiscepto", data)
	return nil
}
func PostSubdiscepto(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &ErrMustLogin{}
	}

	sub := &models.Subdiscepto{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}
	err := db.CreateSubdiscepto(sub, user.ID)
	if err != nil {
		return &ErrInternal{Message: "Error creating subdiscepto", Cause: err}
	}
	http.Redirect(w, r, fmt.Sprintf("/s/%s", sub.Name), http.StatusSeeOther)
	return nil
}
func GetEssay(w http.ResponseWriter, r *http.Request) AppError {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}
	essay, err := db.GetEssay(id)
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	essay.Upvotes, essay.Downvotes, err = db.CountVotes(essay.ID)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	server.RenderHTML(w, "essay", essay)
	return nil
}
