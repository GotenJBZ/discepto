package routes

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/server"
	//"github.com/gorilla/sessions"
)

var cookiestore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

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
	type homeData struct {
		User *models.User
	}
	session, _ := cookiestore.Get(r, "discepto")
	token := session.Values["token"]
	user := &models.User{}
	if u, err := db.GetUserByToken(fmt.Sprintf("%v", token)); err == nil {
		user = u
	}

	server.RenderHTML(w, "home", homeData {
		User: user,
	})
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
	token, err := db.CreateUser(&models.User{
		Name:   r.FormValue("name"),
		Email:  email,
		RoleID: models.RoleAdmin,
	}, r.FormValue("password"))
	if err == db.ErrBadEmailSyntax {
		return &AppError{Cause: err, Message: "Bad email syntax"}
	}
	if err == db.ErrEmailAlreadyUsed {
		return &AppError{Cause: err, Message: "The email is already used"}
	}
	if err != nil {
		return &AppError{Cause: err}
	}

	session, _ := cookiestore.Get(r, "discepto")
	session.Values["token"] = token
	session.Save(r, w)

	http.Redirect(w, r, "/users", http.StatusSeeOther)
	return nil
}
