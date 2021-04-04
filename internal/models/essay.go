package models

import (
	"database/sql"
	"net/url"
	"time"
)

var (
	ReplyTypeSupports = sql.NullString{String: "refutes", Valid: true}
	ReplyTypeRefutes  = sql.NullString{String: "supports", Valid: true}
	ReplyTypeCorrects = sql.NullString{String: "corrects", Valid: true}
	ReplyTypeGeneral  = sql.NullString{String: "general", Valid: true}
)

var AvailableReplyTypes = []sql.NullString{
	ReplyTypeSupports,
	ReplyTypeRefutes,
	ReplyTypeCorrects,
	ReplyTypeGeneral,
}

type EssayPreview struct {
	Thesis    string
	Content   string
	Upvotes   int
	Downvotes int
	Published time.Time
	PostedIn  string
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
	InReplyTo      sql.NullInt32  `db:"in_reply_to"`
	ReplyType      sql.NullString `db:"reply_type"`
	Upvotes        int
	Downvotes      int
}
