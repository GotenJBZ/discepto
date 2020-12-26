package models

import (
	"net/url"
	"time"
)

type Essay struct {
	ID             int
	Thesis         string
	Content        string
	AttributedToID int `db:"attributed_to"`
	Published      time.Time
	Tags           []string
	Sources        []*url.URL
}
