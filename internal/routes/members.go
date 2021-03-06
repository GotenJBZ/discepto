package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gitlab.com/ranfdev/discepto/internal/models"
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
	r.Get("/", routes.renderMembers)
	r.Post("/{userID}", routes.assignRole)
	r.Delete("/{userID}/{roleName}", routes.unassignRole)
}

func (routes *Routes) renderMembers(w http.ResponseWriter, r *http.Request) {
	roleManager := GetRoleManager(r)
	roles, err := roleManager.ListRoles(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	members, err := roleManager.ListMembers(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	data := struct {
		Members []models.Member
		Roles   []models.Role
	}{
		Members: members,
		Roles:   roles,
	}
	routes.tmpls.RenderHTML(w, "members", data)
}
