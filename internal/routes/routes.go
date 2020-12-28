package routes

import (
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/server"
)

type AppError struct {
	Message string
	Status  int
	Cause   error
}

func AppHandler(handler func(w http.ResponseWriter, r *http.Request) *AppError) http.HandlerFunc {
	res := func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)

		if err == nil {
			return
		}
		if err.Status == 0 {
			err.Status = http.StatusInternalServerError
		}
		if err.Message == "" {
			err.Message = "Internal server error"
		}
		http.Error(w, err.Message, http.StatusInternalServerError)
		hlog.FromRequest(r).
			Error().
			Str("request_id", middleware.GetReqID(r.Context())).
			Err(err.Cause).
			Msg(err.Message)
	}
	return res
}
func GetHome(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "home", nil)
}
func GetUsers(w http.ResponseWriter, r *http.Request) *AppError {
	users, err := db.ListUsers()
	if err != nil {
		return &AppError{Cause: err}
	}

	server.RenderHTML(w, "users", users)
	return nil
}
func GetSignup(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "signup", nil)
}
func PostSignup(w http.ResponseWriter, r *http.Request) *AppError {
	email := r.FormValue("email")
	err := db.CreateUser(&models.User{
		Name:   r.FormValue("name"),
		Email:  email,
		RoleID: models.RoleAdmin,
	})
	if err == db.ErrBadEmailSyntax {
		return &AppError{Cause: err, Message: "Bad email syntax"}
	}
	if err == db.ErrEmailAlreadyUsed {
		return &AppError{Cause: err, Message: "The email is already used"}
	}
	if err != nil {
		return &AppError{Cause: err}
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
	return nil
}
