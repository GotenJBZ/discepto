package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/server"
)

func EssaysRouter(r chi.Router) {
	r.Get("/{id}", GetEssay)
	r.Post("/", AppHandler(PostEssay))
	r.Put("/", UpdateEssay)
	r.Delete("/{id}", DeleteEssay)
}
func GetNewEssay(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "newEssay", nil)
}
func GetEssay(w http.ResponseWriter, r *http.Request) {
	essay := models.Essay{
		Thesis:         "asdf",
		Content:        "asdf",
		AttributedToID: 0,
		Published:      time.Now(),
	}

	server.RenderHTML(w, "essay", essay)
}
func PostEssay(w http.ResponseWriter, r *http.Request) *AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &AppError{Message: "Must login to execute this action"}
	}

	essay := models.Essay{
		Thesis:         r.FormValue("thesis"),
		Content:        r.FormValue("content"),
		Tags:           strings.Fields(r.FormValue("tags")),
		AttributedToID: user.ID,
		PostedIn:       r.FormValue("postedIn"),
	}
	err := db.CreateEssay(&essay)
	if err != nil {
		return &AppError{Cause: err}
	}
	http.Redirect(w, r, fmt.Sprintf("/s/%s", essay.PostedIn), http.StatusSeeOther)
	return nil
}
func DeleteEssay(w http.ResponseWriter, r *http.Request) {

}
func UpdateEssay(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Nope")
}
