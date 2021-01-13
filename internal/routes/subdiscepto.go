package routes

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/server"
)

func SubdisceptoRouter(r chi.Router) {
	r.Get("/{name}", AppHandler(GetSubdiscepto))
	r.Post("/", AppHandler(PostSubdiscepto))
}
func GetSubdiscepto(w http.ResponseWriter, r *http.Request) *AppError {
	name := chi.URLParam(r, "name")
	sub, err := db.GetSubdiscepto(name)
	if err != nil {
		return &AppError{Cause: err, Message: fmt.Sprintf("Community %s not found", name)}
	}
	essays, err := db.ListEssays(name)
	if err != nil {
		return &AppError{Cause: err, Message: "Can't list essays"}
	}
	data := struct {
		Name        string
		Description string
		Essays      []models.Essay
	}{
		Name:        sub.Name,
		Description: sub.Description,
		Essays:      essays,
	}
	server.RenderHTML(w, "subdiscepto", data)
	return nil
}
func PostSubdiscepto(w http.ResponseWriter, r *http.Request) *AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &AppError{Message: "Must login to execute this action"}
	}

	sub := &models.Subdiscepto{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}
	err := db.CreateSubdiscepto(sub, user.ID)
	if err != nil {
		return &AppError{Message: "Error creating subdiscepto", Cause: err}
	}
	http.Redirect(w, r, fmt.Sprintf("/s/%s", sub.Name), http.StatusSeeOther)
	return nil
}
