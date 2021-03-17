package db

import (
	"context"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type EssayH struct {
	essayPerms models.EssayPerms
	id         int
	userH      *UserH
	sharedDB   *pgxpool.Pool
}

func (h EssayH) isOwner() bool {
	if h.userH == nil {
		return false
	}
	sql, args, _ := psql.
		Select("1").
		From("essays").
		Where(sq.Eq{"id": h.id, "attributed_to_id": h.userH.userID}).
		ToSql()

	isOwner := 0
	row := h.sharedDB.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&isOwner)
	if err != nil {
		return false
	}

	return isOwner == 1
}
func (h EssayH) GetEssay() (*models.Essay, error) {
	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{"id": h.id}).
		ToSql()

	var essay models.Essay
	err := pgxscan.Get(context.Background(), h.sharedDB, &essay, sql, args...)
	if err != nil {
		return nil, err
	}

	sql, args, _ = psql.
		Select("tag").
		From("essay_tags").
		Where(sq.Eq{"essay_id": h.id}).
		ToSql()
	err = pgxscan.Select(context.Background(), h.sharedDB, &essay.Tags, sql, args...)
	if err != nil {
		return nil, err
	}

	return &essay, nil
}
func (h EssayH) CountVotes() (upvotes, downvotes int, err error) {
	sql, args, _ := psql.
		Select("vote_type", "COUNT(*)").
		From("votes").
		Where(sq.Eq{"essay_id": h.id}).
		GroupBy("vote_type").
		ToSql()

	rows, err := h.sharedDB.Query(context.Background(), sql, args...)
	if err != nil {
		return 0, 0, err
	}
	for rows.Next() {
		var voteType string
		var count int
		err := rows.Scan(&voteType, &count)
		if voteType == string(models.VoteTypeUpvote) {
			upvotes = count
		} else {
			downvotes = count
		}
		if err != nil {
			return 0, 0, err
		}
	}

	return upvotes, downvotes, nil
}
func (h EssayH) DeleteVote() error {
	if h.userH == nil {
		return ErrPermDenied
	}
	sql, args, _ := psql.
		Delete("votes").
		Where(sq.Eq{"user_id": h.userH.userID, "essay_id": h.id}).
		ToSql()
	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func (h EssayH) CreateVote(vote *models.Vote) error {
	if h.userH == nil {
		return ErrPermDenied
	}
	sql, args, _ := psql.
		Insert("votes").
		Columns("user_id", "essay_id", "vote_type").
		Values(vote.UserID, vote.EssayID, vote.VoteType).
		ToSql()

	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	return err
}
func (h EssayH) DeleteEssay() error {
	if !h.essayPerms.DeleteEssay {
		return ErrPermDenied
	}
	return h.deleteEssay()
}
func (h EssayH) deleteEssay() error {
	sql, args, _ := psql.Delete("essays").Where(sq.Eq{"id": h.id}).ToSql()
	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	return err
}
