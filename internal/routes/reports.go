package routes

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/domain"
)

func (routes *Routes) SubReportsRouter(r chi.Router) {
	r.Get("/", routes.AppHandler(routes.GetReports))
	r.Delete("/{reportID}", routes.AppHandler(routes.DeleteReport))
}
func (routes *Routes) GetReports(w http.ResponseWriter, r *http.Request) AppError {
	subH := GetSubdisceptoH(r)
	reports, err := subH.ListReports(r.Context())
	if err != nil {
		fmt.Println(subH.Perms(), err)
		return &ErrInternal{Cause: err}
	}
	routes.tmpls.RenderHTML(w, "reports", struct {
		Reports  []domain.ReportView
		SubPerms domain.SubPerms
	}{
		reports,
		subH.Perms(),
	})
	return nil
}
func (routes *Routes) DeleteReport(w http.ResponseWriter, r *http.Request) AppError {
	subH := GetSubdisceptoH(r)
	reportID, err := strconv.Atoi(chi.URLParam(r, "reportID"))
	if err != nil {
		return &ErrBadRequest{Cause: err}
	}
	err = subH.DeleteReport(r.Context(), reportID)
	fmt.Println(err)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	return routes.GetReports(w, r)
}
