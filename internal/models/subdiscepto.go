package models

type SubdisceptoReq struct {
	Name              string
	Description       string
	MinLength         int
	QuestionsRequired bool
	Public            bool
	Nsfw              bool
}
type Subdiscepto struct {
	Name              string
	Description       string
	RoledomainID      RoleDomain
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
