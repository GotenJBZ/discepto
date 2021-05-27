package db

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type notificationService struct {
	db DBTX
}

func NewNotificationService(db DBTX) *notificationService {
	return &notificationService{
		db,
	}
}

func (s *notificationService) Send(ctx context.Context, notif *models.Notification, toUserID int) error {
	sql, args, _ := psql.
		Insert("notifications").
		Columns("user_id", "notif_type", "title", "text", "action_url").
		Values(toUserID, notif.NotifType, notif.Title, notif.Text, notif.ActionURL.String()).
		ToSql()
	_, err := s.db.Exec(ctx, sql, args...)
	return err
}

func (s *notificationService) List(ctx context.Context, userID int) ([]models.NotifView, error) {
	notifs := []models.NotifView{}
	sql, args, _ := psql.Select("id", "notif_type", "title", "text", "action_url").
		From("notifications").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("id DESC").
		ToSql()

	err := pgxscan.Select(ctx, s.db, &notifs, sql, args...)
	if err != nil {
		return nil, err
	}
	return notifs, nil
}

func (s *notificationService) Delete(ctx context.Context, userID int, notifID int) error {
	sql, args, _ := psql.Delete("notifications").
	Where(sq.Eq{"user_id": userID, "id": notifID}).
		ToSql()

	_, err := s.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
