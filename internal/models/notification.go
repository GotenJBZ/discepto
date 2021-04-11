package models

import "net/url"

type Notification struct {
	Text        string
	UserID      int
	NotifType   string
	Description string
	ActionURL   url.URL `db:"action_url"`
}
