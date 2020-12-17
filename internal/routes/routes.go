package routes

import (
	"net/http"
	"gitlab.com/ranfdev/discepto/internal/utils"
)

func GetHome(w http.ResponseWriter, r *http.Request) {
	utils.RenderHTML(w, "home", nil)
}
