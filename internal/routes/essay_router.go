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
	r.Get("/", AppHandler(GetEssays))
	r.Get("/{id}", GetEssay)
	r.Post("/", AppHandler(PostEssay))
	r.Put("/", UpdateEssay)
	r.Delete("/{id}", DeleteEssay)
}
func GetNewEssay(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "newEssay", nil)
}
func GetEssays(w http.ResponseWriter, r *http.Request) *AppError {
	essays, err := db.ListEssays()
	server.RenderHTML(w, "essays", essays)
	if err != nil {
		return &AppError{Cause: err}
	}
	return nil
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
	session, _ := cookiestore.Get(r, "discepto")
	token := session.Values["token"]

	user, err := db.GetUserByToken(fmt.Sprintf("%v", token))
	if err != nil {
		return &AppError{Cause: err}
	}

	essay := models.Essay{
		Thesis:         r.FormValue("thesis"),
		Content:        r.FormValue("content"),
		Tags:           strings.Fields(r.FormValue("tags")),
		AttributedToID: user.ID,
	}
	err = db.CreateEssay(&essay)
	if err != nil {
		return &AppError{Cause: err}
	}
	http.Redirect(w, r, "/essays", http.StatusSeeOther)
	return nil
}
func DeleteEssay(w http.ResponseWriter, r *http.Request) {

}
func UpdateEssay(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Nope")
}
