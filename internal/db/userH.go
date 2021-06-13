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

func (sdb SharedDB) GetUnsafeUserH(ctx context.Context, userID int) (UserH, error) {
	uH := UserH{
		id:       userID,
		sharedDB: sdb.db,
		perms: userPerms{
			Read:   true,
			Delete: true,
		},
	}
	return uH, nil
}
func (h UserH) ID() int {
	return h.id
}
func (h UserH) Read(ctx context.Context) (*models.User, error) {
	if !h.perms.Read {
		return nil, models.ErrPermDenied
	}
	user := &models.User{}
	sql, args, _ := psql.
		Select("users.name", "users.id", "users.email").
		From("users").
		Where(sq.Eq{"id": h.id}).
		ToSql()

	err := pgxscan.Get(
		ctx,
		h.sharedDB, user,
		sql, args...)

	if err != nil {
		return nil, err
	}
	return user, nil
}
func (h UserH) Delete(ctx context.Context) error {
	if !h.perms.Delete {
		return models.ErrPermDenied
	}
	return h.deleteUser(ctx)
}
func (h *UserH) deleteUser(ctx context.Context) error {
	sql, args, _ := psql.Delete("users").Where(sq.Eq{"id": h.id}).ToSql()
	_, err := h.sharedDB.Exec(ctx, sql, args...)
	return err
}
func listUserEssays(ctx context.Context, db DBTX, userID int) ([]models.EssayView, error) {
	essayPreviews := []models.EssayView{}
	sql, args, _ := selectEssayWithJoins.
		Join("subdisceptos ON subdisceptos.name = essays.posted_in").
		GroupBy("essays.id", "essay_replies.from_id", "users.name").
		Where(sq.Eq{"subdisceptos.public": true, "users.id": userID}).
		OrderBy("essays.id DESC").
		ToSql()

	err := pgxscan.Select(ctx, db, &essayPreviews, sql, args...)
	if err != nil {
		return nil, err
	}
	return essayPreviews, nil
}
