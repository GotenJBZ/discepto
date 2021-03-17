package db

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type userPerms struct {
	Delete bool
}
type UserH struct {
	userID   int
	perms    userPerms
	sharedDB *pgxpool.Pool
}

func (sdb SharedDB) GetUserH(token string) (UserH, error) {
	sql, args, _ := psql.
		Select("user_id").
		From("tokens").
		Where(sq.Eq{"token": token}).
		ToSql()

	userH := UserH{
		sharedDB: sdb.db,
		perms: userPerms{
			Delete: true,
		},
	}
	row := sdb.db.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&userH.userID)

	if err != nil {
		return userH, err
	}
	return userH, nil
}
func (h *UserH) ID() int {
	return h.userID
}
func (h *UserH) Read() (*models.User, error) {
	user := &models.User{}
	sql, args, _ := psql.
		Select("users.name", "users.id", "users.email").
		From("users").
		Where(sq.Eq{"id": h.userID}).
		ToSql()

	err := pgxscan.Get(
		context.Background(),
		h.sharedDB, user,
		sql, args...)

	if err != nil {
		return nil, err
	}
	return user, nil
}
func (h *UserH) Delete() error {
	if !h.perms.Delete {
		return ErrPermDenied
	}
	return h.deleteUser()
}
func (h *UserH) deleteUser() error {
	sql, args, _ := psql.Delete("users").Where(sq.Eq{"id": h.userID}).ToSql()
	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	return err
}
