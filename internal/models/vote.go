package models

type VoteType string

const (
	VoteTypeUpvote   VoteType = "upvote"
	VoteTypeDownvote VoteType = "downvote"
)

type Vote struct {
	UserID   int `db:"user_id"`
	EssayID  int `db:"essay_id"`
	VoteType VoteType
}
