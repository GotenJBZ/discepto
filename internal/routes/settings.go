package routes

import (
	"net/http"

	"github.com/go-chi/chi"
)

func (routes *Routes) SubSettingsRouter(r chi.Router) {
	r.Get("/", routes.AppHandler(routes.GetSubSettings))
}
func (routes *Routes) GlobalSettingsRouter(r chi.Router) {
	r.Get("/", routes.AppHandler(routes.GetGlobalSettings))
}
func (routes *Routes) GetSubSettings(w http.ResponseWriter, r *http.Request) AppError {
	routes.tmpls.RenderHTML(w, "subsettings", nil)
	return nil
}
func (routes *Routes) GetGlobalSettings(w http.ResponseWriter, r *http.Request) AppError {
	routes.tmpls.RenderHTML(w, "subsettings", nil)
	return nil
}
