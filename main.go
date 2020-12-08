package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

var DEBUG bool
var templates *template.Template

type User struct {
	name  string
	email string
	image url.URL
}

// https://www.w3.org/TR/activitystreams-vocabulary/#dfn-article
type Article struct {
	ID           *url.URL
	Name         string // title
	Content      string
	AttributedTo *url.URL
	Published    time.Time
}

func init() {
	templates = template.Must(template.ParseGlob("web/templates/*"))
	if os.Getenv("DEBUG") == "true" {
		DEBUG = true
	} else {
		DEBUG = false
	}
}
func renderHTML(w http.ResponseWriter, tmplName string, data interface{}) {
	t := getTemplates()
	buff := bytes.NewBuffer([]byte{})
	err := t.ExecuteTemplate(buff, tmplName, data)
	if err != nil && tmplName != "404" {
		renderHTML(w, "404", nil)
		log.Println(err)
		return
	}
	w.Header().Add("Content-Type", "text/html")
	w.Write(buff.Bytes())
}
func getTemplates() *template.Template {
	// Reload templates every time when developing locally.
	if DEBUG {
		return template.Must(template.ParseGlob("web/templates/*"))
	} else {
		// Use templates already in memory when in production (faster)
		return templates
	}
}
func getCreateArticle(w http.ResponseWriter, r *http.Request) {
	renderHTML(w, "createArticle", nil)
}
func getArticle(w http.ResponseWriter, r *http.Request) {

	// Create mock article
	id, err1 := url.Parse("https://palavy.com/article/12")
	attributedTo, err2 := url.Parse("https://palavy.com/user/12")
	if err1 != nil || err2 != nil {
		http.NotFound(w, r)
		return
	}
	article := Article{
		ID:           id,
		Name:         "asdf",
		Content:      "asdf",
		AttributedTo: attributedTo,
		Published:    time.Now(),
	}

	renderHTML(w, "article", article)
}
func postArticle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.NotFound(w, r)
		log.Println(err)
		return
	}
	content := r.FormValue("content")
	fmt.Println("content", content)
	w.Write([]byte("lul"))
}
func deleteArticle(w http.ResponseWriter, r *http.Request) {

}
func updateArticle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Nope")
}
func getHome(w http.ResponseWriter, r *http.Request) {
	renderHTML(w, "home", nil)
}
func main() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	// Serve static files
	staticFileServer := http.FileServer(http.Dir("web/static"))
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Requested static")
		fs := http.StripPrefix("/static", staticFileServer)
		fs.ServeHTTP(w, r)
	})

	// Serve dynamic routes
	r.Get("/", getHome)

	r.Route("/article", func(r chi.Router) {
		r.Get("/", getCreateArticle)
		r.Get("/{id}", getArticle)
		r.Post("/", postArticle)
		r.Put("/", updateArticle)
		r.Delete("/{id}", deleteArticle)
	})

	log.Println("Starting server at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
