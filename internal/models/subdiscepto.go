package models

type Subdiscepto struct {
	Name              string
	Description       string
	MinLength         int
	QuestionsRequired bool
	Nsfw              bool
	Public            bool
}
