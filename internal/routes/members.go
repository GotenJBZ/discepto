package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/domain"
)

func (routes *Routes) GlobalMembersRouter(r chi.Router) {
	r.Use(RoleManagerCtx(GetRoleManagerDiscepto))
	routes.membersRouter(r)
}
func (routes *Routes) SubMembersRouter(r chi.Router) {
	r.Use(RoleManagerCtx(GetRoleManagerSubdiscepto))
	routes.membersRouter(r)
}
func (routes *Routes) membersRouter(r chi.Router) {
	r.Get("/", routes.AppHandler(routes.renderMembers))
	r.Post("/{userID}", routes.AppHandler(routes.assignRole))
	r.Delete("/{userID}/{roleName}", routes.AppHandler(routes.unassignRole))
}

func (routes *Routes) renderMembers(w http.ResponseWriter, r *http.Request) AppError {
	roleManager := GetRoleManager(r)
	roles, err := roleManager.ListRoles(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	members, err := roleManager.ListMembers(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}
	data := struct {
		Members []domain.Member
		Roles   []domain.Role
	}{
		Members: members,
		Roles:   roles,
	}
	routes.tmpls.RenderHTML(w, "members", data)
	return nil
}
