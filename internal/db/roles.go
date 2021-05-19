package db

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/ranfdev/discepto/internal/models"
)

var (
	RoleDisceptoAdmin  = models.Role{ID: -123, Name: "admin", Preset: true}
	RoleDisceptoCommon = models.Role{ID: -100, Name: "common", Preset: true}
)

func listRoles(ctx context.Context, db DBTX, domain string) ([]models.Role, error) {
	sql, args, _ := psql.Select("id", "name", "preset").
		From("roles").
		Where(sq.Eq{"domain": domain}).
		ToSql()

	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	roles := []models.Role{}
	for rows.Next() {
		role := models.Role{}
		err := rows.Scan(&role.ID, &role.Name, &role.Preset)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, err
}
func listUserRoles(ctx context.Context, db DBTX, userID int, domain string) ([]models.Role, error) {
	sql, args, _ := psql.Select("id", "name", "preset").
		From("roles").
		Join("user_roles ON roles.id = user_roles.role_id").
		Where(sq.Eq{"domain": domain, "user_id": userID}).
		ToSql()

	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	roles := []models.Role{}
	for rows.Next() {
		role := models.Role{}
		err := rows.Scan(&role.ID, &role.Name, &role.Preset)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, err
}
func findRoleByName(ctx context.Context, db DBTX, domain string, name string) (*models.Role, error) {
	sql, args, _ := psql.Select("id", "name", "preset").
		From("roles").
		Where(sq.Eq{"domain": domain, "name": name}).
		ToSql()

	row := db.QueryRow(ctx, sql, args...)
	role := models.Role{}
	err := row.Scan(&role.ID, &role.Name, &role.Preset)
	if err != nil {
		return nil, err
	}
	role.Domain = domain
	return &role, nil
}
func listRolePerms(ctx context.Context, db DBTX, roleID int) (map[string]bool, error) {
	sql, args, _ := psql.Select("permission").
		From("role_perms").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()

	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	roles := map[string]bool{}
	for rows.Next() {
		perm := ""
		err := rows.Scan(&perm)
		if err != nil {
			return nil, err
		}
		roles[perm] = true
	}
	return roles, err
}

func getUserPerms(ctx context.Context, db DBTX, domain string, userID int) (map[string]bool, error) {
	sql, args, _ := psql.Select("permission").
		From("user_roles").
		Join("role_perms ON user_roles.role_id = role_perms.role_id").
		Join("roles ON user_roles.role_id = roles.id").
		Where(sq.Eq{"domain": domain, "user_id": userID}).
		ToSql()

	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	res := map[string]bool{}
	for rows.Next() {
		perm := ""
		err := rows.Scan(&perm)
		if err != nil {
			return nil, err
		}
		res[perm] = true
	}
	return res, nil
}

func assignRole(ctx context.Context, db DBTX, userID int, roleID int) error {
	sql, args, _ := psql.Insert("user_roles").Columns("user_id", "role_id").Values(userID, roleID).ToSql()
	_, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func unassignRole(ctx context.Context, db DBTX, userID int, roleID int) error {
	sql, args, _ := psql.Delete("user_roles").Where(sq.Eq{"user_id": userID, "role_id": roleID}).ToSql()
	_, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}

func createRole(ctx context.Context, db DBTX, domain string, name string, preset bool, m map[string]bool) (int, error) {
	rowID := -1
	err := execTx(ctx, db, func(ctx context.Context, tx DBTX) error {
		sql, args, _ := psql.
			Insert("roles").
			Columns("domain", "name", "preset").
			Values(domain, name, preset).
			Suffix("RETURNING id").
			ToSql()

		row := tx.QueryRow(ctx, sql, args...)
		err := row.Scan(&rowID)
		if err != nil {
			return err
		}
		err = setPermissions(ctx, tx, rowID, m)
		if err != nil {
			return err
		}

		return nil
	})
	return rowID, err
}
func setPermissions(ctx context.Context, db DBTX, roleID int, perms map[string]bool) error {
	sql, args, _ := psql.
		Delete("role_perms").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()

	_, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	q := psql.
		Insert("role_perms").
		Columns("role_id", "permission")

	for perm, v := range perms {
		if v {
			q = q.Values(roleID, perm)
		}
	}
	sql, args, _ = q.ToSql()
	_, err = db.Exec(ctx, sql, args...)
	return err
}

type HigherRole map[string]bool

func isLowerRole(oldRole HigherRole, newRole map[string]bool) bool {
	for k := range newRole {
		if v, ok := oldRole[k]; !ok || !v {
			return false
		}
	}
	return true
}
func deleteRole(ctx context.Context, db DBTX, roleID int) error {
	sql, args, _ := psql.Delete("roles").Where(sq.Eq{"id": roleID}).ToSql()

	_, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
