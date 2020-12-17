package models

import "time"
type Banned struct {
	ID int
	Start time.Time
	End time.Time
	Motivation Motivation
	Explanation string
	Subdiscepto Subdiscepto
}
