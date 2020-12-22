package routes

import (
	"net/http"

	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/utils"
)

func GetHome(w http.ResponseWriter, r *http.Request) {
	utils.RenderHTML(w, "home", nil)
}
func GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := db.ListUsers()
	if err != nil {
		panic(err)
	}

	utils.RenderHTML(w, "users", users)
}
func GetRegister(w http.ResponseWriter, r *http.Request) {
	utils.RenderHTML(w, "register", nil)
}
func PostRegister(w http.ResponseWriter, r *http.Request) {
	db.CreateUser(&models.User {
		Name: r.FormValue("name"),
		Email: r.FormValue("email"),
		RoleID: models.RoleAdmin,
	})
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}
