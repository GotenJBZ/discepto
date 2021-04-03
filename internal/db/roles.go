package db

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func getGlobalUserPerms(ctx context.Context, db DBTX, userID int) (*models.GlobalPerms, error) {
	perms := models.GlobalPerms{}
	sql, args, _ := psql.Select(
		bool_or("login"),
		bool_or("create_subdiscepto"),
		bool_or("ban_user_globally"),
		bool_or("delete_user"),
		bool_or("manage_global_role"),
		bool_or("read_subdiscepto"),
		bool_or("edit_subdiscepto"),
		bool_or("create_essay"),
		bool_or("delete_essay"),
		bool_or("ban_user"),
		bool_or("delete_subdiscepto"),
		bool_or("change_ranking"),
		bool_or("manage_role"),
	).
		From("user_global_roles").
		Join("global_perms ON user_global_roles.global_perms_id = global_perms.id").
		Join("sub_perms ON global_perms.sub_perms_id = sub_perms.id").
		Where(sq.Eq{"user_id": userID}).
		Having("COUNT(*) > 0").
		ToSql()

	row := db.QueryRow(ctx, sql, args...)
	err := row.Scan(
		&perms.Login,
		&perms.CreateSubdiscepto,
		&perms.BanUserGlobally,
		&perms.DeleteUser,
		&perms.ManageGlobalRole,
		&perms.ReadSubdiscepto,
		&perms.UpdateSubdiscepto,
		&perms.CreateEssay,
		&perms.DeleteEssay,
		&perms.BanUser,
		&perms.DeleteSubdiscepto,
		&perms.ChangeRanking,
		&perms.ManageRole,
	)
	if err != nil {
		return nil, err
	}
	return &perms, nil
}

// Returns the permissions corresponding to a user inside a subdiscepto.
// The user may have multiple roles and may also have a global role,
// granting him permissions inside every subdiscepto.
// We simply fetch all the roles assigned to a user, get the corresponding permissions
// id and UNION the results.
// With the aggregate function "bool_or" we sum every premission.
// The result is 1 row with the correct permissions.
func getSubUserPerms(ctx context.Context, db DBTX, subdiscepto string, userID int) (perms *models.SubPerms, err error) {
	sql, args, _ := psql.
		Select(
			bool_or("read_subdiscepto"),
			bool_or("edit_subdiscepto"),
			bool_or("create_essay"),
			bool_or("delete_essay"),
			bool_or("ban_user"),
			bool_or("delete_subdiscepto"),
			bool_or("change_ranking"),
			bool_or("manage_role"),
		).
		From("user_sub_roles").
		Join("sub_perms ON sub_perms.id = user_sub_roles.sub_perms_id").
		Where(sq.Eq{"subdiscepto": subdiscepto, "user_id": userID}).
		Having("COUNT(*) > 0").
		ToSql()

	row := db.QueryRow(ctx, sql, args...)
	perms = &models.SubPerms{}
	err = row.Scan(
		&perms.ReadSubdiscepto,
		&perms.UpdateSubdiscepto,
		&perms.CreateEssay,
		&perms.DeleteEssay,
		&perms.BanUser,
		&perms.DeleteSubdiscepto,
		&perms.ChangeRanking,
		&perms.ManageRole,
	)
	if err == pgx.ErrNoRows {
		return perms, nil // Return empty perms
	} else if err != nil {
		return nil, err
	}
	return perms, nil
}
func getGlobalRolePerms(ctx context.Context, db DBTX, role string, preset bool) (*models.GlobalPerms, error) {
	sql, args, _ := psql.
		Select("*").
		From("user_global_roles").
		Join("global_perms ON global_perms.id = global_perms_id").
		Join("sub_perms ON global_perms.sub_perms_id = sub_perms.id").
		Where(sq.Eq{"name": role, "preset": preset}).
		ToSql()

	perms := models.GlobalPerms{}
	err := pgxscan.Get(ctx, db, &perms, sql, args...)
	if err != nil {
		return nil, err
	}

	return &perms, err
}

func getSubRolePerms(ctx context.Context, db DBTX, subPermsID int) (*models.SubPerms, error) {
	sql, args, _ := psql.
		Select(
			"read_subdiscepto",
			"edit_subdiscepto",
			"create_essay",
			"delete_essay",
			"ban_user",
			"delete_subdiscepto",
			"change_ranking",
			"manage_role",
		).
		From("sub_perms").
		Where(sq.Eq{"id": subPermsID}).
		ToSql()

	perms := models.SubPerms{}
	err := pgxscan.Get(ctx, db, &perms, sql, args...)
	if err != nil {
		return nil, err
	}

	return &perms, err
}
func assignGlobalRole(ctx context.Context, tx DBTX, assignByUser *int, assignToUser int, role string, preset bool) error {
	q := `
INSERT INTO user_global_roles (assigned_by, user_id, global_perms_id)
SELECT $1, $2, global_perms_id
FROM global_roles
WHERE name = $3 AND preset = $4
`
	byUser := sql.NullInt32{}
	if assignByUser != nil {
		byUser.Int32 = int32(*assignByUser)
	}
	_, err := tx.Exec(ctx, q, byUser, assignToUser, role, preset)
	return err
}
func assignSubRole(ctx context.Context, db DBTX, sub string, assignByUser *int, assignToUser int, subPermsID int) error {
	byUser := sql.NullInt32{}
	if assignByUser != nil {
		byUser.Int32 = int32(*assignByUser)
	}
	sql, args, _ := psql.Insert("user_sub_roles").
		Columns("subdiscepto", "assigned_by", "user_id", "sub_perms_id").
		Values(sub, assignByUser, assignToUser, subPermsID).
		ToSql()
	_, err := db.Exec(ctx, sql, args...)
	return err
}
func unassignSubRole(ctx context.Context, db DBTX, sub string, userID, subPermsID int) error {
	sql, args, _ := psql.Delete("user_sub_roles").
		Where(sq.Eq{"subdiscepto": sub, "user_id": userID, "sub_perms_id": subPermsID}).
		ToSql()
	_, err := db.Exec(ctx, sql, args...)
	return err
}
func createGlobalPerms(ctx context.Context, db DBTX, perms models.GlobalPerms) (int, error) {
	subPermsID, err := createSubPerms(ctx, db, perms.SubPerms)
	if err != nil {
		return 0, err
	}

	sql, args, _ := psql.
		Insert("global_perms").
		Columns(
			"login",
			"create_subdiscepto",
			"ban_user_globally",
			"delete_user",
			"manage_global_role",
			"sub_perms_id",
		).
		Values(
			perms.Login,
			perms.CreateSubdiscepto,
			perms.BanUserGlobally,
			perms.DeleteUser,
			perms.ManageGlobalRole,
			subPermsID,
		).
		Suffix("RETURNING id").
		ToSql()

	row := db.QueryRow(ctx, sql, args...)
	globalPermsID := 0
	err = row.Scan(&globalPermsID)
	return globalPermsID, err
}
func createSubPerms(ctx context.Context, db DBTX, perms models.SubPerms) (int, error) {
	sql, args, _ := psql.
		Insert("sub_perms").
		Columns(
			"read_subdiscepto",
			"edit_subdiscepto",
			"create_essay",
			"delete_essay",
			"ban_user",
			"change_ranking",
			"delete_subdiscepto",
			"manage_role",
		).
		Values(
			perms.ReadSubdiscepto,
			perms.UpdateSubdiscepto,
			perms.CreateEssay,
			perms.DeleteEssay,
			perms.BanUser,
			perms.ChangeRanking,
			perms.DeleteSubdiscepto,
			perms.ManageRole,
		).
		Suffix("RETURNING id").
		ToSql()

	row := db.QueryRow(ctx, sql, args...)
	id := 0
	err := row.Scan(&id)
	return id, err
}
func subPermsIDByRoleName(ctx context.Context, db DBTX, name string, preset bool) (int, error) {
	row := db.QueryRow(ctx, "SELECT sub_perms_id FROM sub_roles WHERE name = $1 AND preset = $2", name, preset)
	id := 0
	err := row.Scan(&id)
	return id, err
}
func createGlobalRole(ctx context.Context, db DBTX, globalPermsID int, name string, preset bool) error {
	sql, args, _ := psql.
		Insert("global_roles").
		Columns("global_perms_id", "name", "preset").
		Values(globalPermsID, name, preset).
		ToSql()

	_, err := db.Exec(ctx, sql, args...)
	return err
}
func createSubRole(ctx context.Context, db DBTX, subPermsID int, subdiscepto string, name string, preset bool) error {
	sql, args, _ := psql.
		Insert("sub_roles").
		Columns("sub_perms_id", "subdiscepto", "name", "preset").
		Values(subPermsID, subdiscepto, name, preset).
		ToSql()

	_, err := db.Exec(ctx, sql, args...)
	return err
}
