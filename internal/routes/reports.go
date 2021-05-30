package routes

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func (routes *Routes) SubReportsRouter(r chi.Router) {
	r.Get("/", routes.GetReports)
	r.Delete("/{reportID}", routes.DeleteReport)
}
func (routes *Routes) GetReports(w http.ResponseWriter, r *http.Request) {
	subH := GetSubdisceptoH(r)
	reports, err := subH.ListReports(r.Context())
	if err != nil {
		fmt.Println(subH.Perms(), err)
		routes.HandleErr(w, r, err)
		return
	}
	routes.tmpls.RenderHTML(w, "reports", struct {
		Reports  []models.ReportView
		SubPerms models.SubPerms
	}{
		reports,
		subH.Perms(),
	})
	return
}
func (routes *Routes) DeleteReport(w http.ResponseWriter, r *http.Request) {
	subH := GetSubdisceptoH(r)
	reportID, err := strconv.Atoi(chi.URLParam(r, "reportID"))
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	err = subH.DeleteReport(r.Context(), reportID)
	fmt.Println(err)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	routes.GetReports(w, r)
	return
}
