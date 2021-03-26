package models

type FlagType string

const (
	FlagTypeOffensive  FlagType = "offensive"
	FlagTypeFake       FlagType = "fake"
	FlagTypeSpam       FlagType = "spam"
	FlagTypeInaccurate FlagType = "inaccurate"
)

type Report struct {
	ID          int
	Flag        FlagType
	Description string
	EssayID     *int `db:"essay_id"` // pointer because it can be null
	FromUserID  int  `db:"from_user_id"`
}
