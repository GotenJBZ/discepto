package adapters

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/ranfdev/discepto/internal/domain"
)


var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
type rbacRepo struct {
	db DBTX
}

func NewRBACRepo(db DBTX) domain.RBACRepo {
	return &rbacRepo {db}
}

func (r *rbacRepo) ListRoles(ctx context.Context, roleDomain string) ([]domain.Role, error) {
	sql, args, _ := psql.Select("id", "name", "preset").
		From("roles").
		Where(sq.Eq{"domain": roleDomain}).
		ToSql()

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	roles := []domain.Role{}
	for rows.Next() {
		role := domain.Role{}
		err := rows.Scan(&role.ID, &role.Name, &role.Preset)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, err
}
func (r *rbacRepo) ListUserRoles(ctx context.Context, userID int, roleDomain string) ([]domain.Role, error) {
	sql, args, _ := psql.Select("id", "name", "preset").
		From("roles").
		Join("user_roles ON roles.id = user_roles.role_id").
		Where(sq.Eq{"domain": roleDomain, "user_id": userID}).
		ToSql()

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	roles := []domain.Role{}
	for rows.Next() {
		role := domain.Role{}
		err := rows.Scan(&role.ID, &role.Name, &role.Preset)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, err
}
func (r *rbacRepo) FindRoleByName(ctx context.Context, roleDomain string, name string) (*domain.Role, error) {
	sql, args, _ := psql.Select("id", "name", "preset").
		From("roles").
		Where(sq.Eq{"domain": roleDomain, "name": name}).
		ToSql()

	row := r.db.QueryRow(ctx, sql, args...)
	role := domain.Role{}
	err := row.Scan(&role.ID, &role.Name, &role.Preset)
	if err != nil {
		return nil, err
	}
	role.Domain = roleDomain
	return &role, nil
}
func (r *rbacRepo) ListRolePerms(ctx context.Context, roleID int) (map[string]bool, error) {
	sql, args, _ := psql.Select("permission").
		From("role_perms").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()

	rows, err := r.db.Query(ctx, sql, args...)
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

func (r *rbacRepo) GetUserPerms(ctx context.Context, roleDomain string, userID int) (map[string]bool, error) {
	sql, args, _ := psql.Select("permission").
		From("user_roles").
		Join("role_perms ON user_roles.role_id = role_perms.role_id").
		Join("roles ON user_roles.role_id = roles.id").
		Where(sq.Eq{"domain": roleDomain, "user_id": userID}).
		ToSql()

	rows, err := r.db.Query(ctx, sql, args...)
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

func (r *rbacRepo) AssignRole(ctx context.Context, userID int, roleID int) error {
	sql, args, _ := psql.Insert("user_roles").Columns("user_id", "role_id").Values(userID, roleID).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func (r *rbacRepo) UnassignRole(ctx context.Context, userID int, roleID int) error {
	sql, args, _ := psql.Delete("user_roles").Where(sq.Eq{"user_id": userID, "role_id": roleID}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}

func (r *rbacRepo) CreateRole(ctx context.Context, role domain.Role, m map[string]bool) (int, error) {
	rowID := -1
	err := execTx(ctx, r.db, func(ctx context.Context, tx DBTX) error {
		sql, args, _ := psql.
			Insert("roles").
			Columns("domain", "name", "preset").
			Values(role.Domain, role.Name, role.Preset).
			Suffix("RETURNING id").
			ToSql()

		row := tx.QueryRow(ctx, sql, args...)
		err := row.Scan(&rowID)
		if err != nil {
			return err
		}
		err = r.SetPermissions(ctx, rowID, m)
		if err != nil {
			return err
		}

		return nil
	})
	return rowID, err
}
func (r *rbacRepo) SetPermissions(ctx context.Context, roleID int, perms map[string]bool) error {
	sql, args, _ := psql.
		Delete("role_perms").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()

	_, err := r.db.Exec(ctx, sql, args...)
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
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *rbacRepo) DeleteRole(ctx context.Context, roleID int) error {
	sql, args, _ := psql.Delete("roles").Where(sq.Eq{"id": roleID}).ToSql()

	_, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
