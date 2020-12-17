package models

type VoteType int;
const (
	Upvote VoteType = 1
	Downvote VoteType = -1
)
type Vote struct {
	From User
	Essay Essay
	VoteType VoteType
}
