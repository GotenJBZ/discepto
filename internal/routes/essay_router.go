package routes

import (
	"fmt"
	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/server"
	"gitlab.com/ranfdev/discepto/internal/db"
	"log"
	"net/http"
	"time"
)

func EssayRouter(r chi.Router) {
	r.Get("/", GetCreateEssay)
	r.Get("/{id}", GetEssay)
	r.Post("/", PostEssay)
	r.Put("/", UpdateEssay)
	r.Delete("/{id}", DeleteEssay)
}
func GetCreateEssay(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "createEssay", nil)
}
func GetEssay(w http.ResponseWriter, r *http.Request) {
	users, err := db.ListUsers()
	if err != nil {
		panic(err)
	}

	essay := models.Essay{
		Thesis:       "asdf",
		Content:      "asdf",
		AttributedTo: users[0],
		Published:    time.Now(),
	}

	server.RenderHTML(w, "essay", essay)
}
func PostEssay(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.NotFound(w, r)
		log.Println(err)
		return
	}
	content := r.FormValue("content")
	w.Write([]byte("Thanks for answering with " + content + ". Your answer is going to be discarded anyway"))
}
func DeleteEssay(w http.ResponseWriter, r *http.Request) {

}
func UpdateEssay(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Nope")
}
