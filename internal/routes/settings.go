package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/domain"
)

func (routes *Routes) SubSettingsRouter(r chi.Router) {
	r.Get("/", routes.AppHandler(routes.GetSubSettings))
}
func (routes *Routes) GlobalSettingsRouter(r chi.Router) {
	r.Get("/", routes.AppHandler(routes.GetGlobalSettings))
}
func (routes *Routes) GetSubSettings(w http.ResponseWriter, r *http.Request) AppError {
	subH := GetSubdisceptoH(r)
	sub, err := subH.ReadRaw(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	routes.tmpls.RenderHTML(w, "subsettings", struct{ Subdiscepto *domain.Subdiscepto }{sub})
	return nil
}
func (routes *Routes) GetGlobalSettings(w http.ResponseWriter, r *http.Request) AppError {
	routes.tmpls.RenderHTML(w, "subsettings", nil)
	return nil
}
