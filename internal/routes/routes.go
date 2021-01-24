package routes

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/server"
)

var cookiestore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

func UserCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := cookiestore.Get(r, "discepto")
		token := session.Values["token"]
		token = fmt.Sprintf("%v", token) // conv to string

		if token == "" {
			ctx := context.WithValue(r.Context(), "user", nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		user, err := db.GetUserByToken(fmt.Sprintf("%v", token))
		if err != nil {
			session.Values["token"] = ""
			session.Save(r, w)
			ctx := context.WithValue(r.Context(), "user", nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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
func GetHome(w http.ResponseWriter, r *http.Request) *AppError {
	type homeData struct {
		User           *models.User
		LoggedIn       bool
		MySubdisceptos []string
		RecentEssays   []*models.Essay
	}
	user, ok := r.Context().Value("user").(*models.User)

	data := homeData{User: user, LoggedIn: ok}
	if data.LoggedIn {
		mySubs, err := db.ListMySubdisceptos(user.ID)
		if err != nil {
			return &AppError{Message: "Can't list joined communities", Cause: err}
		}
		data.MySubdisceptos = mySubs

		recentEssays, err := db.ListRecentEssaysIn(mySubs)
		if err != nil {
			return &AppError{Message: "Can't list recent essays", Cause: err}
		}
		data.RecentEssays = recentEssays
	}

	server.RenderHTML(w, "home", data)
	return nil
}
func GetUsers(w http.ResponseWriter, r *http.Request) *AppError {
	users, err := db.ListUsers()
	if err != nil {
		return &AppError{Cause: err}
	}

	server.RenderHTML(w, "users", users)
	return nil
}
func GetSignout(w http.ResponseWriter, r *http.Request) *AppError {
	session, _ := cookiestore.Get(r, "discepto")
	token := session.Values["token"]

	// Remove token before deleting from db, to signout in any case
	session.Values["token"] = ""
	session.Save(r, w)

	err := db.Signout(fmt.Sprintf("%v", token))
	if err != nil {
		return &AppError{Cause: err}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func GetSignup(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "signup", nil)
}
func GetLogin(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "login", nil)
}
func GetNewSubdiscepto(w http.ResponseWriter, r *http.Request) {
	server.RenderHTML(w, "newSubdiscepto", nil)
}
func PostLogin(w http.ResponseWriter, r *http.Request) *AppError {
	token, err := db.Login(r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		return &AppError{
			Cause:   err,
			Message: "Bad email or password",
		}
	}
	session, _ := cookiestore.Get(r, "discepto")
	session.Values["token"] = token
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func PostSignup(w http.ResponseWriter, r *http.Request) *AppError {
	email := r.FormValue("email")
	err := db.CreateUser(&models.User{
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

	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return nil
}
