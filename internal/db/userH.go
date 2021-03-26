package db

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type userPerms struct {
	Delete bool
	Read   bool
}
type UserH struct {
	id       int
	perms    userPerms
	sharedDB DBTX
}

func ToUserH(t interface{}) (*UserH, bool) {
	v, ok := t.(*UserH)
	return v, ok
}

func (sdb SharedDB) GetUserH(token string) (UserH, error) {
	sql, args, _ := psql.
		Select("user_id").
		From("tokens").
		Where(sq.Eq{"token": token}).
		ToSql()

	uH := UserH{
		sharedDB: sdb.db,
		perms: userPerms{
			Read:   true,
			Delete: true,
		},
	}
	row := sdb.db.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&uH.id)

	if err != nil {
		return uH, err
	}
	return uH, nil
}
func (h UserH) ID() int {
	return h.id
}
func (h UserH) Read() (*models.User, error) {
	if !h.perms.Read {
		return nil, ErrPermDenied
	}
	user := &models.User{}
	sql, args, _ := psql.
		Select("users.name", "users.id", "users.email").
		From("users").
		Where(sq.Eq{"id": h.id}).
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
func (h UserH) Delete() error {
	if !h.perms.Delete {
		return ErrPermDenied
	}
	return h.deleteUser()
}
func (h UserH) JoinSub(subH SubdisceptoH) error {
	return subH.addMember(h)
}
func (h UserH) LeaveSub(subH SubdisceptoH) error {
	return subH.removeMember(h)
}
func (h *UserH) deleteUser() error {
	sql, args, _ := psql.Delete("users").Where(sq.Eq{"id": h.id}).ToSql()
	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	return err
}
func (h UserH) ListMySubdisceptos() (subs []string, err error) {
	sql, args, _ := psql.
		Select("subdiscepto").
		From("subdiscepto_users").
		Where(sq.Eq{"user_id": h.id}).
		ToSql()

	err = pgxscan.Select(context.Background(), h.sharedDB, &subs, sql, args...)
	if err != nil {
		return nil, err
	}
	return subs, nil
}
