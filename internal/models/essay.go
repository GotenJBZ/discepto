package models

import (
	"database/sql"
	"net/url"
	"time"
)

const (
	ReplyTypeSupports = "supports"
	ReplyTypeRefutes  = "refutes"
	ReplyTypeCorrects = "corrects"
	ReplyTypeGeneral  = "general"
)

var AvailableReplyTypes = []string{
	ReplyTypeSupports,
	ReplyTypeRefutes,
	ReplyTypeCorrects,
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
