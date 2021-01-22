package models

import (
	"database/sql"
	"net/url"
	"time"
)

func ParseReplyType(s string) int {
	switch s {
	case "inFavor":
		return 1
	case "against":
		return -1
	}
	return 0
}

type Essay struct {
	ID             int
	Thesis         string
	Content        string
	AttributedToID int `db:"attributed_to_id"`
	Published      time.Time
	Tags           []string
	Sources        []*url.URL
	PostedIn       string
	InReplyTo      sql.NullInt32 `db:"in_reply_to"`
	ReplyType      int           `db:"reply_type"`
	Upvotes        int
	Downvotes      int
}
