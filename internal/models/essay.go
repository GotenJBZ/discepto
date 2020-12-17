package models

import "time"
type Essay struct {
	ID int
	Thesis string
	Content string
	AttributedTo User
	Published time.Time
	Tags []Tag `gorm:"many2many:essay_tags"`
	Source []Source `gorm:"many2many:essay_sources"`
}
