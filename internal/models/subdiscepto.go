package models

type Subdiscepto struct {
	Name              string
	Description       string
	MinLength         int
	QuestionsRequired bool
	Nsfw              bool
	Public            bool
}

type SubdisceptoView struct {
	Name         string
	Description  string
	MembersCount int
	IsMember     bool
}
