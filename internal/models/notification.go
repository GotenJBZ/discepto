package models

import (
	"context"
	"net/url"
)

type NotifType string

const (
	NotifTypeReply  = "reply"
	NotifTypeUpvote = "upvote"
)

type Notification struct {
	NotifType string
	Title     string
	Text      string
	ActionURL url.URL `db:"action_url"`
}
type NotifView struct {
	ID        int
	NotifType string
	Title     string
	Text      string
	ActionURL string
}

type NotificationService interface {
	Send(ctx context.Context, notif *Notification, toUserID int) error
	List(ctx context.Context, userID int) ([]NotifView, error)
	Delete(ctx context.Context, userID int, id int) error
}
