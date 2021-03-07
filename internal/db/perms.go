package db

// This file is crucial. Before editing, be sure to understand how it works
// ----
// Discepto uses a complex Role Based Access Control (RBAC).
// Each user can have one or multiple roles.
// Each role gives a set of permissions.
// When retrieving roles, permissions get summed (boolean OR): the permissions set to true always win.
// Be sure to see the first database migration in the folder /migrations to see how this is implemented
// in the database

// ## Global roles
// They override any local role. The first user registered on Discepto has complete control over
// the platform (meaning it gets a global role with full permissions).

// ## Local roles
// They are local to a subdiscepto. Discepto has some preset local roles, but also supports the
// creations of custom ones.

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func bool_or(col string) string {
	return fmt.Sprintf("bool_or(%s) AS %s", col, col)
}

// Returns the permissions corresponding to a user inside a subdiscepto.
// The user may have multiple roles:
// - preset (already defined when Discepto is installed)
// - custom (defined by community admins at runtime)
// The user may have a global role, granting him permissions inside every subdiscepto.
// That means we have 3 tables to check:
// - user_preset_sub_roles
// - user_custom_sub_roles
// - user_global_roles
// We simply fetch all the roles assigned to a user, get the corresponding permission row
// and UNION the results. Then we use the aggregate function "bool_or" to sum
// every premission. The result is 1 row with the correct permissions.
func (db *DB) GetSubPerms(userID int, subName string) (perms models.SubPerms, err error) {
	// TODO: Check global roles

	queryPresetSubRoles := sq.Select("sub_perms_id").
		From("user_preset_sub_roles").
		Join("preset_sub_roles ON user_preset_sub_roles.role_name = preset_sub_roles.name").
		Where(sq.Eq{"user_preset_sub_roles.subdiscepto": subName, "user_id": userID})

	queryCustomSubPerms := sq.Select("sub_perms_id").
		From("user_custom_sub_roles").
		Join("custom_sub_roles ON user_custom_sub_roles.role_name = custom_sub_roles.name AND user_custom_sub_roles.subdiscepto = custom_sub_roles.subdiscepto").
		Where(sq.Eq{"custom_sub_roles.subdiscepto": subName, "user_id": userID})

	queryAllSubPerms := queryPresetSubRoles.Suffix("UNION").SuffixExpr(queryCustomSubPerms)

	sql, args, _ := psql.
		Select(
			bool_or("create_essay"),
			bool_or("delete_essay"),
			bool_or("ban_user"),
			bool_or("change_ranking"),
			bool_or("delete_subdiscepto"),
			bool_or("add_mod"),
		).
		FromSelect(queryAllSubPerms, "user_perms_ids").
		Join("sub_perms ON sub_perms.id = user_perms_ids.sub_perms_id").
		PlaceholderFormat(sq.Dollar).
		ToSql()

	err = pgxscan.Get(context.Background(), db.db, &perms, sql, args...)
	return perms, err
}
