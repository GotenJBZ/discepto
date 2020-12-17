package models

type Motivation int;
const (
	Insulting Motivation = iota
	Spam
)
type Report struct {
	ID int
	Motivation Motivation
	Comment string
	SpecificEssay Essay
	FromUser User
	ToUser User
}
