package db

import (
	"context"

	"github.com/georgysavva/scany/pgxscan"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type EssayH struct {
	sharedDB   DBTX
	id         int
	essayPerms models.EssayPerms
}

func isEssayOwner(ctx context.Context, db DBTX, essayID int, userID int) bool {
	sql, args, _ := psql.
		Select("1").
		From("essays").
		Where(sq.Eq{"id": essayID, "attributed_to_id": userID}).
		ToSql()

	isOwner := 0
	row := db.QueryRow(ctx, sql, args...)
	err := row.Scan(&isOwner)
	if err != nil {
		return false
	}

	return isOwner == 1
}
func (h EssayH) GetEssay(ctx context.Context) (*models.Essay, error) {
	if !h.essayPerms.Read {
		return nil, ErrPermDenied
	}
	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{"id": h.id}).
		ToSql()

	var essay models.Essay
	err := pgxscan.Get(ctx, h.sharedDB, &essay, sql, args...)
	if err != nil {
		return nil, err
	}

	sql, args, _ = psql.
		Select("tag").
		From("essay_tags").
		Where(sq.Eq{"essay_id": h.id}).
		ToSql()
	err = pgxscan.Select(ctx, h.sharedDB, &essay.Tags, sql, args...)
	if err != nil {
		return nil, err
	}

	return &essay, nil
}
func (h EssayH) CreateReport(ctx context.Context, rep models.Report, userH UserH) error {
	if !h.essayPerms.Read {
		return ErrPermDenied
	}
	if rep.EssayID != &h.id || rep.FromUserID != userH.id {
		return ErrPermDenied
	}
	sql, args, _ := psql.
		Insert("reports").
		Columns("flag", "essay_id", "from_user_id", "description").
		Values(rep.Flag, h.id, userH.id, rep.Description).
		Suffix("RETURNING id").
		ToSql()

	row := h.sharedDB.QueryRow(ctx, sql, args...)
	err := row.Scan(&rep.ID)
	if err != nil {
		return err
	}
	return nil
}
func (h EssayH) CountVotes(ctx context.Context) (upvotes, downvotes int, err error) {
	if !h.essayPerms.Read {
		return 0, 0, ErrPermDenied
	}
	sql, args, _ := psql.
		Select("vote_type", "COUNT(*)").
		From("votes").
		Where(sq.Eq{"essay_id": h.id}).
		GroupBy("vote_type").
		ToSql()

	rows, err := h.sharedDB.Query(ctx, sql, args...)
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
func (h EssayH) DeleteVote(ctx context.Context, uH UserH) error {
	if !h.essayPerms.Read {
		return ErrPermDenied
	}
	sql, args, _ := psql.
		Delete("votes").
		Where(sq.Eq{"user_id": uH.id, "essay_id": h.id}).
		ToSql()

	_, err := h.sharedDB.Exec(ctx, sql, args...)
	return err
}
func (h EssayH) CreateVote(ctx context.Context, uH UserH, vote models.VoteType) error {
	if !h.essayPerms.Read {
		return ErrPermDenied
	}
	sql, args, _ := psql.
		Insert("votes").
		Columns("user_id", "essay_id", "vote_type").
		Values(uH.id, h.id, vote).
		ToSql()

	_, err := h.sharedDB.Exec(ctx, sql, args...)
	return err
}
func (h EssayH) DeleteEssay(ctx context.Context) error {
	if !h.essayPerms.DeleteEssay {
		return ErrPermDenied
	}
	return h.deleteEssay(ctx)
}
func (h EssayH) deleteEssay(ctx context.Context) error {
	sql, args, _ := psql.Delete("essays").Where(sq.Eq{"id": h.id}).ToSql()
	_, err := h.sharedDB.Exec(ctx, sql, args...)
	return err
}
