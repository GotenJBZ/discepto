package routes

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/httprate"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/render"
	"gitlab.com/ranfdev/discepto/web"
)

type disceptoCtxKey int

const (
	UserHCtxKey disceptoCtxKey = iota
	DisceptoHCtxKey
	SubdisceptoHCtxKey
	EssayHCtxKey
)

const (
	LimitGlobalCount, LimitGlobalDuration = 180, 1 * time.Minute
	LimitPostCount, LimitPostDuration     = 30, 1 * time.Minute
)

type Routes struct {
	envConfig      *models.EnvConfig
	db             *db.SharedDB
	tmpls          *render.Templates
	sessionManager *scs.SessionManager
}

func NewRouter(config *models.EnvConfig, db *db.SharedDB, log zerolog.Logger, tmpls *render.Templates) chi.Router {
	sessionManager := NewSessionManager(config)
	routes := &Routes{envConfig: config, db: db, tmpls: tmpls, sessionManager: sessionManager}
	sessionManager.ErrorFunc = func(w http.ResponseWriter, r *http.Request, err error) {
		routes.RenderErr(w, r, &ErrInternal{Cause: err})
	}

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

	r.Use(sessionManager.LoadAndSave)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	if !config.Debug {
		r.Use(httprate.LimitByIP(LimitGlobalCount, LimitGlobalDuration))
		r.Use(func(next http.Handler) http.Handler {
			limiter := httprate.Limit(LimitPostCount, LimitPostDuration, httprate.KeyByIP)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" || r.Method == "DELETE" || r.Method == "PUT" {
					limiter(next)
				} else {
					next.ServeHTTP(w, r)
				}
			})
		})
	}

	// Try retrieving basic user data for every request
	r.Use(routes.UserCtx)
	// Try retrieving discepto handler for every request
	r.Use(routes.DisceptoCtx)

	// Serve static files
	var staticFileFS fs.FS
	if config.Debug {
		staticFileFS = os.DirFS("./web/static")
	} else {
		staticFileFS, _ = fs.Sub(web.FS, "static")
	}
	staticFileServer := http.FileServer(http.FS(staticFileFS))
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/static", staticFileServer)
		fs.ServeHTTP(w, r)
	})

	// Serve dynamic routes
	r.Get("/", routes.AppHandler(routes.GetHome))
	r.Get("/signup", routes.GetSignup)
	r.Post("/signup", routes.AppHandler(routes.PostSignup))
	r.Get("/login", routes.GetLogin)
	r.Post("/login", routes.AppHandler(routes.PostLogin))
	r.Route("/s", routes.SubdisceptoRouter)

	loggedIn := r.With(routes.EnforceCtx(UserHCtxKey))
	loggedIn.Get("/u", routes.AppHandler(routes.GetUserSelf))
	loggedIn.Get("/u/{viewingUserID}", routes.AppHandler(routes.GetUser))
	loggedIn.Post("/signout", routes.AppHandler(routes.PostSignout))
	loggedIn.Get("/newessay", routes.AppHandler(routes.GetNewEssay))
	loggedIn.Post("/newessay", routes.AppHandler(routes.PostEssay))
	loggedIn.Route("/roles", routes.GlobalRolesRouter)
	loggedIn.Route("/members", routes.GlobalMembersRouter)
	loggedIn.Route("/settings", routes.GlobalSettingsRouter)
	loggedIn.Get("/search", routes.AppHandler(routes.GetSearch))
	loggedIn.Get("/newsubdiscepto", routes.GetNewSubdiscepto)
	loggedIn.Get("/notifications", routes.AppHandler(routes.GetNotifications))
	loggedIn.Post("/notifications/{notifID}", routes.AppHandler(routes.ViewDeleteNotif))

	// Fallback
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		routes.tmpls.RenderHTML(w, "404", nil)
	})
	return r
}
func NewSessionManager(config *models.EnvConfig) *scs.SessionManager {
	sessionManager := scs.New()
	sessionManager.Lifetime = 30 * 24 * time.Hour
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.SameSite = http.SameSiteStrictMode
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.Secure = !config.Debug
	return sessionManager
}

func (routes *Routes) UserCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !routes.sessionManager.Exists(r.Context(), "userID") {
			next.ServeHTTP(w, r)
			return
		}

		userID := routes.sessionManager.GetInt(r.Context(), "userID")
		userH, err := routes.db.GetUnsafeUserH(r.Context(), userID)
		if err != nil {
			routes.sessionManager.Clear(r.Context())
		}

		ctx := context.WithValue(r.Context(), UserHCtxKey, &userH)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (routes *Routes) DisceptoCtx(next http.Handler) http.Handler {
	return routes.AppHandler(func(w http.ResponseWriter, r *http.Request) AppError {
		userH := GetUserH(r)

		disceptoH, err := routes.db.GetDisceptoH(r.Context(), userH)
		if err != nil {
			return &ErrInternal{Cause: err}
		}

		ctx := context.WithValue(r.Context(), DisceptoHCtxKey, disceptoH)
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
func GetUserH(r *http.Request) *db.UserH {
	h, _ := r.Context().Value(UserHCtxKey).(*db.UserH)
	return h
}
func GetSubdisceptoH(r *http.Request) *db.SubdisceptoH {
	h, _ := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)
	return h
}
func GetDisceptoH(r *http.Request) *db.DisceptoH {
	h, _ := r.Context().Value(DisceptoHCtxKey).(*db.DisceptoH)
	return h
}
func GetEssayH(r *http.Request) *db.EssayH {
	h, _ := r.Context().Value(EssayHCtxKey).(*db.EssayH)
	return h
}

func LimitPost() {
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
	routes.tmpls.RenderHTML(w, "500", err.Message)
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
			routes.RenderErr(w, r, err)
		}
	}
	return res
}
func (routes *Routes) RenderErr(w http.ResponseWriter, r *http.Request, err AppError) {
	loggableErr := err.Respond(w, r, routes)

	hlog.FromRequest(r).
		Error().
		Str("request_id", middleware.GetReqID(r.Context())).
		Err(loggableErr.Cause).
		Send()
}

// Routes
func (routes *Routes) GetHome(w http.ResponseWriter, r *http.Request) AppError {
	type homeData struct {
		LoggedIn       bool
		MySubdisceptos []string
		RecentEssays   []models.EssayView
	}
	disceptoH := GetDisceptoH(r)
	userH := GetUserH(r)

	data := homeData{LoggedIn: userH != nil}
	if data.LoggedIn {
		mySubs, err := userH.ListMySubdisceptos(r.Context())
		if err != nil {
			return &ErrInternal{Message: "Can't list joined communities", Cause: err}
		}
		data.MySubdisceptos = mySubs

		recentEssays, err := disceptoH.ListRecentEssaysIn(r.Context(), mySubs)

		if err != nil {
			return &ErrInternal{Message: "Can't list recent essays", Cause: err}
		}
		data.RecentEssays = recentEssays
	}

	routes.tmpls.RenderHTML(w, "home", data)
	return nil
}
func (routes *Routes) PostSignout(w http.ResponseWriter, r *http.Request) AppError {
	routes.sessionManager.RenewToken(r.Context())
	routes.sessionManager.Remove(r.Context(), "userID")

	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return nil
}
func (routes *Routes) GetUser(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	disceptoH := GetDisceptoH(r)
	vUserID, err := strconv.Atoi(chi.URLParam(r, "viewingUserID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}
	essays, err := disceptoH.ListUserEssays(r.Context(), vUserID)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	userData, err := disceptoH.ReadPublicUser(r.Context(), vUserID)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	mySubs, err := userH.ListMySubdisceptos(r.Context())
	routes.tmpls.RenderHTML(w, "user", struct {
		User            *models.UserView
		Essays          []models.EssayView
		FilterReplyType string
		MySubdisceptos  []string
	}{
		User:            userData,
		Essays:          essays,
		FilterReplyType: "general",
		MySubdisceptos:  mySubs,
	})
	return nil
}
func (routes *Routes) GetUserSelf(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	disceptoH := GetDisceptoH(r)
	essays, err := disceptoH.ListUserEssays(r.Context(), userH.ID())
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	userData, err := disceptoH.ReadPublicUser(r.Context(), userH.ID())
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	mySubs, err := userH.ListMySubdisceptos(r.Context())
	routes.tmpls.RenderHTML(w, "user", struct {
		User            *models.UserView
		Essays          []models.EssayView
		FilterReplyType string
		MySubdisceptos  []string
	}{
		User:            userData,
		Essays:          essays,
		FilterReplyType: "general",
		MySubdisceptos:  mySubs,
	})
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
func (routes *Routes) GetNotifications(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	notifs, err := userH.ListNotifications(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	routes.tmpls.RenderHTML(w, "notifications", notifs)
	return nil
}
func (routes *Routes) ViewDeleteNotif(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	notifID, err := strconv.Atoi(chi.URLParam(r, "notifID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}
	err = userH.DeleteNotif(r.Context(), notifID)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	actionURL := r.URL.Query().Get("action_url")
	w.Header().Add("HX-Redirect", actionURL)
	http.Redirect(w, r, actionURL, http.StatusAccepted)
	return nil
}
func (routes *Routes) PostLogin(w http.ResponseWriter, r *http.Request) AppError {
	userH, err := routes.db.Login(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		return &ErrBadRequest{
			Cause:      err,
			Motivation: "Bad email or password",
		}
	}
	err = routes.sessionManager.RenewToken(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	routes.sessionManager.Put(r.Context(), "userID", userH.ID())

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
