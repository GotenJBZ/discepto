package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/render"
)

type Routes struct {
	envConfig   *models.EnvConfig
	db          *db.SharedDB
	tmpls       *render.Templates
	cookiestore *sessions.CookieStore
}

func NewRouter(config *models.EnvConfig, db *db.SharedDB, log zerolog.Logger, tmpls *render.Templates) chi.Router {
	cookiestore := sessions.NewCookieStore(config.SessionKey)
	cookiestore.Options = &sessions.Options{
		HttpOnly: true,
	}
	routes := &Routes{envConfig: config, db: db, tmpls: tmpls, cookiestore: cookiestore}
	r := chi.NewRouter()

	logger := hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.
			FromRequest(r).
			Info().
			Str("request_id", middleware.GetReqID(r.Context())).
			Int("status", status).
			Str("url", r.URL.String()).
			Str("method", r.Method).Int("size", size).
			Dur("duration", duration).
			Str("ip", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Msg("")
	})

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(hlog.NewHandler(log))
	r.Use(logger)

	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	// Try retrieving basic user data for every request
	r.Use(routes.UserCtx)

	// Serve static files
	staticFileServer := http.FileServer(http.Dir("./web/static"))
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/static", staticFileServer)
		fs.ServeHTTP(w, r)
	})

	// Serve dynamic routes
	r.Get("/", routes.AppHandler(routes.GetHome))

	r.Get("/users", routes.AppHandler(routes.GetUsers))

	r.Get("/signup", routes.GetSignup)
	r.Post("/signup", routes.AppHandler(routes.PostSignup))

	r.Get("/signout", routes.AppHandler(routes.GetSignout))

	r.Get("/login", routes.GetLogin)
	r.Post("/login", routes.AppHandler(routes.PostLogin))

	r.Get("/newessay", routes.AppHandler(routes.GetNewEssay))
	r.Post("/newessay", routes.AppHandler(routes.PostEssay))

	r.Route("/s", routes.SubdisceptoRouter)
	r.Get("/newsubdiscepto", routes.GetNewSubdiscepto)
	return r
}
func (routes *Routes) UserCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := routes.cookiestore.Get(r, "discepto")
		token := session.Values["token"]
		token = fmt.Sprintf("%v", token) // conv to string

		if token == "" {
			ctx := context.WithValue(r.Context(), "user", nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		user, err := routes.db.GetUserH(r.Context(), fmt.Sprintf("%v", token))
		if err != nil {
			session.Values["token"] = ""
			session.Save(r, w)
			ctx := context.WithValue(r.Context(), "user", nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		ctx := context.WithValue(r.Context(), "user", &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Interface shared by every custom http error.
// Needed to provide custom error handling for each error type
type AppError interface {
	Respond(w http.ResponseWriter, r *http.Request) LoggableErr
}

// Printable error data related to a request
type LoggableErr struct {
	Message string
	Status  int
	Cause   error
}

// Specific errors
type ErrInternal struct {
	Message string
	Cause   error
}

func (err *ErrInternal) Respond(w http.ResponseWriter, r *http.Request) LoggableErr {
	loggableErr := LoggableErr{
		Cause:   err.Cause,
		Message: err.Message,
		Status:  http.StatusInternalServerError,
	}
	http.Error(w, "Internal server error", loggableErr.Status)
	return loggableErr
}

type ErrNotFound struct {
	Cause error
	Thing string
}

func (err *ErrNotFound) Respond(w http.ResponseWriter, r *http.Request) LoggableErr {
	loggableErr := LoggableErr{
		Cause:   err.Cause,
		Status:  http.StatusNotFound,
		Message: fmt.Sprintf("Retrieving %s", err.Thing),
	}
	http.NotFound(w, r)
	return loggableErr
}

type ErrMustLogin struct{}

func (err *ErrMustLogin) Respond(w http.ResponseWriter, r *http.Request) LoggableErr {
	loggableErr := LoggableErr{
		Cause:  errors.New("Not logged in"),
		Status: http.StatusSeeOther,
	}
	http.Redirect(w, r, "/login", loggableErr.Status)
	return loggableErr
}

type ErrBadRequest struct {
	Cause      error
	Motivation string
}

func (err *ErrBadRequest) Respond(w http.ResponseWriter, r *http.Request) LoggableErr {
	loggableErr := LoggableErr{
		Cause:   err.Cause,
		Message: err.Motivation,
		Status:  http.StatusBadRequest,
	}
	http.Error(w, err.Motivation, loggableErr.Status)
	return loggableErr
}

type ErrInsuffPerms struct {
	Action string
}

func (err *ErrInsuffPerms) Respond(w http.ResponseWriter, r *http.Request) LoggableErr {
	loggableErr := LoggableErr{
		Cause:   errors.New(fmt.Sprintf("Insufficient permissions for action %v)", err.Action)),
		Message: "Insufficient permissions to execute this action",
		Status:  http.StatusBadRequest,
	}
	http.Error(w, loggableErr.Message, loggableErr.Status)
	return loggableErr
}

// Wrapper to handle errors returned by routes
func (routes *Routes) AppHandler(handler func(w http.ResponseWriter, r *http.Request) AppError) http.HandlerFunc {
	res := func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err != nil {
			loggableErr := err.Respond(w, r)

			hlog.FromRequest(r).
				Error().
				Str("request_id", middleware.GetReqID(r.Context())).
				Err(loggableErr.Cause).
				Send()
		}
	}
	return res
}

// Routes
func (routes *Routes) GetHome(w http.ResponseWriter, r *http.Request) AppError {
	type homeData struct {
		User           *models.User
		LoggedIn       bool
		MySubdisceptos []string
		RecentEssays   []*models.Essay
	}
	user, ok := r.Context().Value("user").(*db.UserH)

	userData := &models.User{}
	if ok {
		var err error
		userData, err = user.Read(r.Context())
		if err != nil {
			return &ErrInternal{Cause: err}
		}
	}
	fmt.Println(user)
	data := homeData{User: userData, LoggedIn: ok}
	if data.LoggedIn {
		mySubs, err := user.ListMySubdisceptos(r.Context())
		if err != nil {
			return &ErrInternal{Message: "Can't list joined communities", Cause: err}
		}
		data.MySubdisceptos = mySubs

		recentEssays, err := routes.db.ListRecentEssaysIn(r.Context(), mySubs)
		if err != nil {
			return &ErrInternal{Message: "Can't list recent essays", Cause: err}
		}
		data.RecentEssays = recentEssays
	}

	routes.tmpls.RenderHTML(w, "home", data)
	return nil
}
func (routes *Routes) GetUsers(w http.ResponseWriter, r *http.Request) AppError {
	user, _ := r.Context().Value("user").(*db.UserH)
	disceptoH := routes.db.GetDisceptoH(r.Context(), user)

	users, err := disceptoH.ListUsers(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	routes.tmpls.RenderHTML(w, "users", users)
	return nil
}
func (routes *Routes) signOut(w http.ResponseWriter, r *http.Request) error {
	session, _ := routes.cookiestore.Get(r, "discepto")
	token := session.Values["token"]

	// Remove token before deleting from db, to signout in any case
	session.Values["token"] = ""
	session.Save(r, w)

	err := routes.db.Signout(r.Context(), fmt.Sprintf("%v", token))
	return err
}
func (routes *Routes) GetSignout(w http.ResponseWriter, r *http.Request) AppError {
	err := routes.signOut(w, r)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func (routes *Routes) GetSignup(w http.ResponseWriter, r *http.Request) {
	routes.tmpls.RenderHTML(w, "signup", nil)
}
func (routes *Routes) GetLogin(w http.ResponseWriter, r *http.Request) {
	routes.tmpls.RenderHTML(w, "login", nil)
}
func (routes *Routes) GetNewSubdiscepto(w http.ResponseWriter, r *http.Request) {
	routes.tmpls.RenderHTML(w, "newSubdiscepto", nil)
}
func (routes *Routes) PostLogin(w http.ResponseWriter, r *http.Request) AppError {
	token, err := routes.db.Login(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		return &ErrBadRequest{
			Cause:      err,
			Motivation: "Bad email or password",
		}
	}
	session, _ := routes.cookiestore.Get(r, "discepto")
	session.Values["token"] = token
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
func (routes *Routes) PostSignup(w http.ResponseWriter, r *http.Request) AppError {
	email := r.FormValue("email")
	_, err := routes.db.CreateUser(r.Context(), &models.User{
		Name:  r.FormValue("name"),
		Email: email,
	}, r.FormValue("password"))
	if err == db.ErrInvalidFormat {
		return &ErrBadRequest{Cause: err, Motivation: "Bad email syntax"}
	}
	if err == db.ErrEmailAlreadyUsed {
		return &ErrBadRequest{Cause: err, Motivation: "The email is already used"}
	}
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return nil
}
