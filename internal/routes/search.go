package routes

import (
	"fmt"
	"net/http"
	"strings"

	"gitlab.com/ranfdev/discepto/internal/models"
)

func (routes *Routes) GetSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	disceptoH := GetDisceptoH(r)
	userH := GetUserH(r)
	searchBy := r.URL.Query().Get("searchBy")
	query := r.URL.Query().Get("q")
	filterType := r.URL.Query().Get("filterType")

	var essays []models.EssayView
	var err error
	switch searchBy {
	case "thesis":
		essays, err = disceptoH.SearchByThesis(ctx, query)
	case "tags":
		tags := strings.Split(query, ",")
		essays, err = disceptoH.SearchByTags(ctx, tags)
		fmt.Println(tags)
	}
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	mySubs, err := disceptoH.ListUserSubdisceptos(r.Context(), userH)

	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	routes.tmpls.RenderHTML(w, "search", struct {
		Essays         []models.EssayView
		MySubdisceptos []models.SubdisceptoView
		Query          string
		FilterType     string
		SearchBy       string
	}{
		MySubdisceptos: mySubs,
		Essays:         essays,
		Query:          query,
		FilterType:     filterType,
		SearchBy:       searchBy,
	})
}
