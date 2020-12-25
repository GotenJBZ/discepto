package routes

import (
	"net/http"

	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/server"
	"gitlab.com/ranfdev/discepto/internal/utils"
)

func GetHome(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "home", nil)
}
func GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := db.ListUsers()
	if err != nil {
		panic(err)
	}

	server.RenderHTML(w, "users", users)
}
func GetRegister(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "register", nil)
}
func PostRegister(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	if !utils.ValidateEmail(email) {
		http.Error(w, "Invalid email", http.StatusInternalServerError)
		return
	}
	err := db.CreateUser(&models.User {
		Name: r.FormValue("name"),
		Email: email,
		RoleID: models.RoleAdmin,
	})
	if err != nil {
		http.Error(w, "Error, status 500", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}
