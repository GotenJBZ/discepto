package models

import "net/url"

type NotifType string

const (
	NotifTypeReply = "reply"
)

type Notification struct {
	UserID    int
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
