package models

import (
	"database/sql"
	"net/url"
	"time"
)

const (
	ReplyTypeInFavor = 1
	ReplyTypeAgainst = -1
)

func ParseReplyType(s string) int {
	switch s {
	case "inFavor":
		return ReplyTypeInFavor
	case "against":
		return ReplyTypeInFavor
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
