package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func (routes *Routes) GlobalMembersRouter(r chi.Router) {
	r.Get("/", routes.AppHandler(routes.GetGlobalMembers))
	r.Post("/{userID}", routes.AppHandler(routes.assignGlobalRole))
	r.Delete("/{userID}/{roleID}", routes.AppHandler(routes.unassignGlobalRole))
}
func (routes *Routes) SubMembersRouter(r chi.Router) {
	r.Get("/", routes.AppHandler(routes.GetSubMembers))
	r.Post("/{userID}", routes.AppHandler(routes.assignSubRole))
	r.Delete("/{userID}/{roleID}", routes.AppHandler(routes.unassignSubRole))
}

func (routes *Routes) GetGlobalMembers(w http.ResponseWriter, r *http.Request) AppError {
	disceptoH := GetDisceptoH(r)
	members, err := disceptoH.ListMembers(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	roles, err := disceptoH.ListRoles(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	data := struct {
		Subdiscepto *struct{}
		Members     []models.Member
		Roles       []models.Role
	}{
		Subdiscepto: nil,
		Members:     members,
		Roles:       roles,
	}
	routes.tmpls.RenderHTML(w, "members", data)
	return nil
}

func (routes *Routes) GetSubMembers(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	subH := GetSubdisceptoH(r)
	sub, err := subH.ReadView(r.Context(), userH)
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	members, err := subH.ListMembers(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	roles, err := subH.ListRoles(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	data := struct {
		Subdiscepto *models.SubdisceptoView
		Members     []models.Member
		Roles       []models.Role
	}{
		Subdiscepto: sub,
		Members:     members,
		Roles:       roles,
	}
	routes.tmpls.RenderHTML(w, "members", data)
	return nil
}
