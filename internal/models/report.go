package models

type FlagType string

const (
	FlagTypeOffensive  FlagType = "offensive"
	FlagTypeFake       FlagType = "fake"
	FlagTypeSpam       FlagType = "spam"
	FlagTypeInaccurate FlagType = "inaccurate"
)
type Report struct {
	ID int
	Flag FlagType
	Description string
	Essay *Essay
	FromUser *User
	ToUser *User
}
