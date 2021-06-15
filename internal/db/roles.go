package db

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"gitlab.com/ranfdev/discepto/internal/models"
)

var (
	RoleDisceptoAdmin  = models.Role{ID: -123, Name: "admin", Preset: true}
	RoleDisceptoCommon = models.Role{ID: -100, Name: "common", Preset: true}
)

func createRoledomain(ctx context.Context, db DBTX, domainType string) (models.RoleDomain, error) {
	sql, args, _ := psql.
		Insert("roledomains").
		Columns("domain_type").
		Values(domainType).
		Suffix("RETURNING id").
		ToSql()

	row := db.QueryRow(ctx, sql, args...)
	var id int
	err := row.Scan(&id)
	return models.RoleDomain(id), err
}
func listRoles(ctx context.Context, db DBTX, domain models.RoleDomain) ([]models.Role, error) {
	sql, args, _ := psql.Select("id", "name", "preset").
		From("roles").
		Where(sq.Eq{"roledomain_id": domain}).
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
func listUserRoles(ctx context.Context, db DBTX, userID int, domain models.RoleDomain) ([]models.Role, error) {
	sql, args, _ := psql.Select("id", "name", "preset").
		From("roles").
		Join("user_roles ON roles.id = user_roles.role_id").
		Where(sq.Eq{"roledomain_id": domain, "user_id": userID}).
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
func findRoleByName(ctx context.Context, db DBTX, domain models.RoleDomain, name string) (*models.Role, error) {
	sql, args, _ := psql.Select("id", "name", "preset").
		From("roles").
		Where(sq.Eq{"roledomain_id": domain, "name": name}).
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
func listRolePerms(ctx context.Context, db DBTX, roleID int) (models.Perms, error) {
	sql, args, _ := psql.Select("permission").
		From("role_perms").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()

	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	perms := []models.Perm{}
	for rows.Next() {
		perm := ""
		err := rows.Scan(&perm)
		if err != nil {
			return nil, err
		}
		perms = append(perms, models.Perm(perm))
	}
	return models.NewPerms(perms...), err
}

func getUserPerms(ctx context.Context, db DBTX, domain models.RoleDomain, userID int) (models.Perms, error) {
	sql, args, _ := psql.Select("permission").
		From("user_roles").
		Join("role_perms ON user_roles.role_id = role_perms.role_id").
		Join("roles ON user_roles.role_id = roles.id").
		Where(sq.Eq{"roledomain_id": domain, "user_id": userID}).
		ToSql()

	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	arrPerms := []models.Perm{}
	for rows.Next() {
		perm := ""
		err := rows.Scan(&perm)
		if err != nil {
			return nil, err
		}
		arrPerms = append(arrPerms, models.Perm(perm))
	}
	return models.NewPerms(arrPerms...), nil
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
func unassignAll(ctx context.Context, db DBTX, userID int, domain models.RoleDomain) error {
	return execTx(ctx, db, func(ctx context.Context, tx DBTX) error {
		roles, err := listUserRoles(ctx, tx, userID, domain)
		if err != nil {
			return err
		}
		for _, role := range roles {
			err := unassignRole(ctx, tx, userID, role.ID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func createRole(ctx context.Context, db DBTX, role models.Role, m models.Perms) (int, error) {
	rowID := -1
	err := execTx(ctx, db, func(ctx context.Context, tx DBTX) error {
		sql, args, _ := psql.
			Insert("roles").
			Columns("roledomain_id", "name", "preset").
			Values(role.Domain, role.Name, role.Preset).
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
func setPermissions(ctx context.Context, db DBTX, roleID int, perms models.Perms) error {
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

	for perm := range perms {
		q = q.Values(roleID, perm)
	}
	sql, args, _ = q.ToSql()
	_, err = db.Exec(ctx, sql, args...)
	return err
}

func deleteRole(ctx context.Context, db DBTX, roleID int) error {
	sql, args, _ := psql.Delete("roles").Where(sq.Eq{"id": roleID}).ToSql()

	_, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func listDomainsWithPerms(ctx context.Context,
	db DBTX,
	userID int,
	domainType string,
	perms models.Perms) ([]models.RoleDomain, error) {
	sql, args, _ := psql.Select("roledomains.id").
		From("role_perms").
		Join("user_roles ON user_roles.role_id = role_perms.role_id").
		Join("roles ON roles.id = role_perms.role_id").
		Join("roledomains ON roles.roledomain_id = roledomains.id").
		Where(
			sq.Eq{"user_roles.user_id": userID,
				"roledomains.domain_type": domainType,
				"permission":              perms.List()},
		).
		GroupBy("user_roles.user_id", "roledomains.id").
		Having(sq.Eq{"COUNT(DISTINCT permission)": len(perms.List())}).
		ToSql()

	res := []models.RoleDomain{}
	fmt.Println(sql, args)
	err := pgxscan.Select(ctx, db, &res, sql, args...)
	if err != nil {
		return nil, err
	}
	return res, nil
}
