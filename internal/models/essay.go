package models

import (
	"database/sql"
	"net/url"
	"time"
)

const (
	ReplyTypeInFavor    = "in favor"
	ReplyTypeAgainst    = "against"
	ReplyTypeCorrection = "correction"
	ReplyTypeGeneral    = "general"
)

var AvailableReplyTypes = []string{
	ReplyTypeInFavor,
	ReplyTypeAgainst,
	ReplyTypeCorrection,
	ReplyTypeGeneral,
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
	ReplyType      string        `db:"reply_type"`
	Upvotes        int
	Downvotes      int
}
