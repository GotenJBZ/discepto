package routes

import (
	"fmt"
	"net/http"
	"strings"

	"gitlab.com/ranfdev/discepto/internal/domain"
)

func (routes *Routes) GetSearch(w http.ResponseWriter, r *http.Request) AppError {
	ctx := r.Context()
	disceptoH := GetDisceptoH(r)
	userH := GetUserH(r)
	searchBy := r.URL.Query().Get("searchBy")
	query := r.URL.Query().Get("q")
	filterType := r.URL.Query().Get("filterType")

	var essays []domain.EssayView
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
		return &ErrInternal{Cause: err}
	}

	mySubs, err := userH.ListMySubdisceptos(ctx)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	routes.tmpls.RenderHTML(w, "search", struct {
		Essays         []domain.EssayView
		MySubdisceptos []string
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
	return nil
}
