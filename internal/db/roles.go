package db

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func getGlobalPerms(db DBTX, uH *UserH) models.GlobalPerms {
	perms := models.GlobalPerms{}
	if uH != nil {
		sql, args, _ := psql.Select(
			bool_or("login"),
			bool_or("create_subdiscepto"),
			bool_or("ban_user_globally"),
			bool_or("delete_user"),
			bool_or("add_admin"),
		).
			From("user_global_roles").
			Join("global_perms ON user_global_roles.global_perms_id = global_perms.id").
			Where(sq.Eq{"user_id": uH.id}).
			Having("COUNT(*) > 0").
			ToSql()

		row := db.QueryRow(context.Background(), sql, args...)
		row.Scan(
			&perms.Login,
			&perms.CreateSubdiscepto,
			&perms.BanUserGlobally,
			&perms.DeleteUser,
			&perms.AddAdmin,
		)
	}
	return perms
}

// Returns the permissions corresponding to a user inside a subdiscepto.
// The user may have multiple roles and may also have a global role,
// granting him permissions inside every subdiscepto.
// We simply fetch all the roles assigned to a user, get the corresponding permission row
// and UNION the results. Then we use the aggregate function "bool_or" to sum
// every premission. The result is 1 row with the correct permissions.
func getSubPerms(db DBTX, subdiscepto string, uH UserH) (perms *models.SubPerms, err error) {
	// TODO: Check global roles

	querySubRolesPermsID := sq.Select("sub_perms_id").
		From("user_sub_roles").
		Where(sq.Eq{"subdiscepto": subdiscepto, "user_id": uH.id})

	sql, args, _ := psql.
		Select(
			bool_or("create_essay"),
			bool_or("delete_essay"),
			bool_or("ban_user"),
			bool_or("change_ranking"),
			bool_or("delete_subdiscepto"),
			bool_or("add_mod"),
		).
		FromSelect(querySubRolesPermsID, "user_perms_ids").
		Join("sub_perms ON sub_perms.id = user_perms_ids.sub_perms_id").
		PlaceholderFormat(sq.Dollar).
		Having("COUNT(*) > 0").
		ToSql()

	row := db.QueryRow(context.Background(), sql, args...)
	perms = &models.SubPerms{}
	err = row.Scan(
		&perms.CreateEssay,
		&perms.DeleteEssay,
		&perms.BanUser,
		&perms.ChangeRanking,
		&perms.DeleteSubdiscepto,
		&perms.AddMod,
	)
	if err == pgx.ErrNoRows {
		return perms, nil // Return empty perms
	} else if err != nil {
		return nil, err
	}
	// If no error was returned, it means the user has some role assigned.
	// If a user has at least one role, it automatically gets read permissions
	// In fact, admins can ban a user simply by removing all roles from him
	perms.Read = true
	perms.EssayPerms.Read = true
	return perms, nil
}
func assignNamedGlobalRole(tx DBTX, userID int, role string, preset bool) error {
	sql := `
INSERT INTO user_global_roles (user_id, global_perms_id, sub_perms_id)
SELECT $1, global_perms_id, sub_perms_id
FROM global_roles
WHERE name = $2 AND preset = $3
`
	_, err := tx.Exec(context.Background(), sql, userID, role, preset)
	return err
}
func assignNamedSubRole(db DBTX, userID int, sub string, role string, preset bool) error {
	sql := `
INSERT INTO user_sub_roles (subdiscepto, user_id, sub_perms_id)
SELECT $1, $2, sub_perms_id
FROM sub_roles
WHERE (subdiscepto = $3 OR subdiscepto IS NULL) AND name = $4 AND preset = $5
`
	_, err := db.Exec(context.Background(), sql, sub, userID, sub, role, preset)
	return err
}
