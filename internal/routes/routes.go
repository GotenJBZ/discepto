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

type disceptoCtxKey int

const (
	UserHCtxKey disceptoCtxKey = iota
	DiscpetoHCtxKey
	SubdisceptoHCtxKey
	EssayHCtxKey
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
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
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
	// Try retrieving discepto handler for every request
	r.Use(routes.DisceptoCtx)

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
	r.Get("/login", routes.GetLogin)
	r.Post("/login", routes.AppHandler(routes.PostLogin))
	r.Route("/s", routes.SubdisceptoRouter)

	loggedIn := r.With(routes.EnforceCtx(UserHCtxKey))
	loggedIn.Get("/signout", routes.AppHandler(routes.GetSignout))
	loggedIn.Get("/newessay", routes.AppHandler(routes.GetNewEssay))
	loggedIn.Post("/newessay", routes.AppHandler(routes.PostEssay))
	loggedIn.Route("/roles", routes.GlobalRolesRouter)
	loggedIn.Route("/members", routes.GlobalMembersRouter)
	loggedIn.Route("/settings", routes.GlobalSettingsRouter)
	loggedIn.Get("/newsubdiscepto", routes.GetNewSubdiscepto)

	// Fallback
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		routes.tmpls.RenderHTML(w, "404", nil)
	})
	return r
}

func (routes *Routes) UserCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := routes.cookiestore.Get(r, "discepto")
		token := session.Values["token"]
		token = fmt.Sprintf("%v", token) // conv to string

		if token == "" {
			ctx := context.WithValue(r.Context(), UserHCtxKey, nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		user, err := routes.db.GetUserH(r.Context(), fmt.Sprintf("%v", token))
		if err != nil {
			session.Values["token"] = ""
			session.Save(r, w)
			ctx := context.WithValue(r.Context(), UserHCtxKey, nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		ctx := context.WithValue(r.Context(), UserHCtxKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (routes *Routes) DisceptoCtx(next http.Handler) http.Handler {
	return routes.AppHandler(func(w http.ResponseWriter, r *http.Request) AppError {
		userH, _ := r.Context().Value(UserHCtxKey).(*db.UserH)

		subH, err := routes.db.GetDisceptoH(r.Context(), userH)
		if err != nil {
			return &ErrInternal{Cause: err}
		}

		ctx := context.WithValue(r.Context(), DiscpetoHCtxKey, subH)
		next.ServeHTTP(w, r.WithContext(ctx))
		return nil
	})
}
func (routes *Routes) EnforceCtx(ctxValue disceptoCtxKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return routes.AppHandler(func(w http.ResponseWriter, r *http.Request) AppError {
			if r.Context().Value(ctxValue) == nil {
				return &ErrInsuffPerms{}
			}
			next.ServeHTTP(w, r)
			return nil
		})
	}
}

// Interface shared by every custom http error.
// Needed to provide custom error handling for each error type
type AppError interface {
	Respond(w http.ResponseWriter, r *http.Request, routes *Routes) LoggableErr
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

func (err *ErrInternal) Respond(w http.ResponseWriter, r *http.Request, routes *Routes) LoggableErr {
	loggableErr := LoggableErr{
		Cause:   err.Cause,
		Message: err.Message,
		Status:  http.StatusInternalServerError,
	}
	routes.tmpls.RenderHTML(w, "500", nil)
	return loggableErr
}

type ErrNotFound struct {
	Cause error
	Thing string
}

func (err *ErrNotFound) Respond(w http.ResponseWriter, r *http.Request, routes *Routes) LoggableErr {
	loggableErr := LoggableErr{
		Cause:   err.Cause,
		Status:  http.StatusNotFound,
		Message: fmt.Sprintf("Retrieving %s", err.Thing),
	}
	routes.tmpls.RenderHTML(w, "404", nil)
	return loggableErr
}

type ErrMustLogin struct{}

func (err *ErrMustLogin) Respond(w http.ResponseWriter, r *http.Request, routes *Routes) LoggableErr {
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

func (err *ErrBadRequest) Respond(w http.ResponseWriter, r *http.Request, routes *Routes) LoggableErr {
	loggableErr := LoggableErr{
		Cause:   err.Cause,
		Message: err.Motivation,
		Status:  http.StatusBadRequest,
	}
	routes.tmpls.RenderHTML(w, "400", err.Motivation)
	return loggableErr
}

type ErrInsuffPerms struct {
	Action string
}

func (err *ErrInsuffPerms) Respond(w http.ResponseWriter, r *http.Request, routes *Routes) LoggableErr {
	loggableErr := LoggableErr{
		Cause:   errors.New(fmt.Sprintf("Insufficient permissions for action %v)", err.Action)),
		Message: "Insufficient permissions to execute this action",
		Status:  http.StatusBadRequest,
	}
	routes.tmpls.RenderHTML(w, "403", nil)
	return loggableErr
}

// Wrapper to handle errors returned by routes
func (routes *Routes) AppHandler(handler func(w http.ResponseWriter, r *http.Request) AppError) http.HandlerFunc {
	res := func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err != nil {
			loggableErr := err.Respond(w, r, routes)

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
		LoggedIn       bool
		MySubdisceptos []string
		RecentEssays   []models.EssayView
	}
	user, ok := r.Context().Value(UserHCtxKey).(*db.UserH)

	data := homeData{LoggedIn: ok}
	if data.LoggedIn {
		mySubs, err := user.ListMySubdisceptos(r.Context())
		if err != nil {
			return &ErrInternal{Message: "Can't list joined communities", Cause: err}
		}
		data.MySubdisceptos = mySubs

		recentEssays, err := routes.db.ListRecentEssaysIn(r.Context(), mySubs)

		for i := 0; i < len(recentEssays); i++ {
			recentEssays[i].Content = recentEssays[i].Content[0:150] + "..."
		}

		if err != nil {
			return &ErrInternal{Message: "Can't list recent essays", Cause: err}
		}
		data.RecentEssays = recentEssays
	}

	routes.tmpls.RenderHTML(w, "home", data)
	return nil
}
func (routes *Routes) GetUsers(w http.ResponseWriter, r *http.Request) AppError {
	disceptoH := r.Context().Value(DiscpetoHCtxKey).(*db.DisceptoH)

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

	errorMessage := ""
	if err != nil {
		switch err {
		case db.ErrInvalidFormat:
			errorMessage = "Invalid email syntax"
		case db.ErrEmailAlreadyUsed:
			errorMessage = "Email already used"
		case db.ErrWeakPasswd:
			errorMessage =
				`The password is too weak.
The password must:
- Be Longer than 8 characters
- Contain at least 1 number
- Contain at least 1 letter
- Contain at least 1 special character
- Be Smaller than 64 characters
`
		default:
			errorMessage = "Internal error"
		}
		return &ErrBadRequest{Cause: err, Motivation: errorMessage}
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return nil
}
