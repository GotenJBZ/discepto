package db

import (
	"context"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type EssayH struct {
	sharedDB   *pgxpool.Pool
	id         int
	essayPerms models.EssayPerms
}

func isEssayOwner(db *pgxpool.Pool, essayID int, userID int) bool {
	sql, args, _ := psql.
		Select("1").
		From("essays").
		Where(sq.Eq{"id": essayID, "attributed_to_id": userID}).
		ToSql()

	isOwner := 0
	row := db.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&isOwner)
	if err != nil {
		return false
	}

	return isOwner == 1
}
func (h EssayH) GetEssay() (*models.Essay, error) {
	if !h.essayPerms.Read {
		return nil, ErrPermDenied
	}
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
func (h EssayH) DeleteVote(uH UserH) error {
	sql, args, _ := psql.
		Delete("votes").
		Where(sq.Eq{"user_id": uH.id, "essay_id": h.id}).
		ToSql()

	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	return err
}
func (h EssayH) CreateVote(uH UserH, vote models.VoteType) error {
	sql, args, _ := psql.
		Insert("votes").
		Columns("user_id", "essay_id", "vote_type").
		Values(uH.id, h.id, vote).
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
