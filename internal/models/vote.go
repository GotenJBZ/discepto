package models

type VoteType string

const (
	VoteTypeUpvote   VoteType = "upvote"
	VoteTypeDownvote VoteType = "downvote"
)
type Vote struct {
	From User
	Essay Essay
	VoteType VoteType
}
