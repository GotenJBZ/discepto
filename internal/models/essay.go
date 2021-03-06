package models

import (
	"database/sql"
	"errors"
	"net/url"
	"regexp"
	"time"
)

var (
	ErrTooManyTags   = errors.New("too many tags")
	ErrBadContentLen = errors.New("bad content length")
)
var (
	ReplyTypeSupports = sql.NullString{String: "supports", Valid: true}
	ReplyTypeRefutes  = sql.NullString{String: "refutes", Valid: true}
	ReplyTypeCorrects = sql.NullString{String: "corrects", Valid: true}
	ReplyTypeGeneral  = sql.NullString{String: "general", Valid: true}
)

var AvailableReplyTypes = []sql.NullString{
	ReplyTypeSupports,
	ReplyTypeRefutes,
	ReplyTypeCorrects,
	ReplyTypeGeneral,
}

// Represents the "essays" table in the database and strictly related data
type Essay struct {
	ID             int
	Thesis         string
	Content        string
	Published      time.Time
	PostedIn       string
	AttributedToID int `db:"attributed_to_id"`
	Tags           []string
	Sources        []url.URL
	Questions      []Question
	Replying
}

// Preview and View contain data generated by queries
type EssayView struct {
	ID               int
	Thesis           string
	Content          string
	Published        time.Time
	PostedIn         string
	AttributedToID   int `db:"attributed_to_id"`
	AttributedToName string
	Upvotes          int
	Downvotes        int
	Tags             []string
	Replying
}
type EssayRow struct {
	ID               int
	Thesis           string
	Content          string
	Published        time.Time
	PostedIn         string
	AttributedToID   int `db:"attributed_to_id"`
	AttributedToName string
	Upvotes          int
	Downvotes        int
	Tag              string
	Replying
}

type Replying struct {
	InReplyTo sql.NullInt32  `db:"in_reply_to"`
	ReplyType sql.NullString `db:"reply_type"`
}

type MDLink struct {
	Text string
	URL  string
}

func FindMDLinks(content string) []MDLink {
	regex := regexp.MustCompile(`\[([\w\s\d]+)\]\((https?:\/\/[\w\d./?=#]+)\)`)
	matches := regex.FindAllStringSubmatch(content, -1)
	mdLinks := []MDLink{}
	for _, m := range matches {
		link := MDLink{
			Text: m[1],
			URL:  m[2],
		}
		mdLinks = append(mdLinks, link)
	}
	return mdLinks
}
