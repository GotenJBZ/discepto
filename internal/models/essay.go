package models

import "time"

type Essay struct {
	ID           int
	Thesis       string
	Content      string
	AttributedTo User
	Published    time.Time
	Tags         []Tag
	Source       []Source
}
