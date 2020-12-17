package routes

import (
	"fmt"
	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/utils"
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
	utils.RenderHTML(w, "createEssay", nil)
}
func GetEssay(w http.ResponseWriter, r *http.Request) {
	// mock data
	creator := models.User{
		Name:  "pippo",
		Email: "pippo@gmail.com",
	}
	essay := models.Essay{
		Thesis:       "asdf",
		Content:      "asdf",
		AttributedTo: creator,
		Published:    time.Now(),
	}

	utils.RenderHTML(w, "essay", essay)
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
