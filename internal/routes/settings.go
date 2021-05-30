package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func (routes *Routes) SubSettingsRouter(r chi.Router) {
	r.Get("/", routes.GetSubSettings)
}
func (routes *Routes) GlobalSettingsRouter(r chi.Router) {
	r.Get("/", routes.GetGlobalSettings)
}
func (routes *Routes) GetSubSettings(w http.ResponseWriter, r *http.Request) {
	subH := GetSubdisceptoH(r)
	sub, err := subH.ReadRaw(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	routes.tmpls.RenderHTML(w, "subsettings", struct{ Subdiscepto *models.Subdiscepto }{sub})
	return
}
func (routes *Routes) GetGlobalSettings(w http.ResponseWriter, r *http.Request) {
	routes.tmpls.RenderHTML(w, "subsettings", nil)
	return
}
