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
		return nil, ErrPermDenied
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
		return ErrPermDenied
	}
	return h.deleteUser(ctx)
}
func (h *UserH) deleteUser(ctx context.Context) error {
	sql, args, _ := psql.Delete("users").Where(sq.Eq{"id": h.id}).ToSql()
	_, err := h.sharedDB.Exec(ctx, sql, args...)
	return err
}
func (h UserH) ListMySubdisceptos(ctx context.Context) (subs []string, err error) {
	sql, args, _ := psql.
		Select("subdiscepto").
		From("subdiscepto_users").
		Where(sq.Eq{"user_id": h.id, "left_at": nil}).
		ToSql()

	err = pgxscan.Select(ctx, h.sharedDB, &subs, sql, args...)
	if err != nil {
		return nil, err
	}
	return subs, nil
}
func (h UserH) ListNotifications(ctx context.Context) ([]models.NotifView, error) {
	if !h.perms.Read {
		return nil, ErrPermDenied
	}
	notifs := []models.NotifView{}
	sql, args, _ := psql.Select("id", "notif_type", "title", "text", "action_url").
		From("notifications").
		Where(sq.Eq{"user_id": h.id}).
		OrderBy("id DESC").
		ToSql()

	err := pgxscan.Select(ctx, h.sharedDB, &notifs, sql, args...)
	if err != nil {
		return nil, err
	}
	return notifs, nil
}
func (h UserH) DeleteNotif(ctx context.Context, notifID int) error {
	if !h.perms.Read {
		return ErrPermDenied
	}
	sql, args, _ := psql.Delete("notifications").
		Where(sq.Eq{"user_id": h.id, "id": notifID}).
		ToSql()

	_, err := h.sharedDB.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func sendNotification(ctx context.Context, db DBTX, notif models.Notification) error {
	sql, args, _ := psql.
		Insert("notifications").
		Columns("user_id", "notif_type", "title", "text", "action_url").
		Values(notif.UserID, notif.NotifType, notif.Title, notif.Text, notif.ActionURL.String()).
		ToSql()
	_, err := db.Exec(ctx, sql, args...)
	return err
}
func listUserEssays(ctx context.Context, db DBTX, userID int) ([]models.EssayView, error) {
	essayPreviews := []models.EssayView{}
	sql, args, _ := selectEssayWithJoins.
		Join("subdisceptos ON subdisceptos.name = essays.posted_in").
		GroupBy("essays.id", "essay_replies.from_id", "users.name").
		Where(sq.Eq{"subdisceptos.public": true}).
		OrderBy("essays.id DESC").
		ToSql()

	err := pgxscan.Select(ctx, db, &essayPreviews, sql, args...)
	if err != nil {
		return nil, err
	}
	return essayPreviews, nil
}
