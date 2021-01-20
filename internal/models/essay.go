package models

import (
	"net/url"
	"time"
)

type Essay struct {
	ID             int
	Thesis         string
	Content        string
	AttributedToID int `db:"attributed_to_id"`
	Published      time.Time
	Tags           []string
	Sources        []*url.URL
	PostedIn       string
	Upvotes        int
	Downvotes      int
}
