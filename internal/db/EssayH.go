package db

import (
	"context"
	"fmt"
	"net/url"

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

var selectEssay = psql.
	Select(
		"essays.id",
		"essays.thesis",
		"essays.content",
		"essays.attributed_to_id",
		"essays.published",
		"essays.posted_in",
		"SUM(CASE votes.vote_type WHEN 'upvote' THEN 1 ELSE 0 END) AS upvotes",
		"SUM(CASE votes.vote_type WHEN 'downvote' THEN 1 ELSE 0 END) AS downvotes",
		"essay_replies.to_id AS in_reply_to",
		"essay_replies.reply_type AS reply_type",
		"users.name AS attributed_to_name",
	)
var selectEssayWithJoins = selectEssay.
	From("essays").
	LeftJoin("essay_replies ON essay_replies.from_id = essays.id").
	LeftJoin("votes ON votes.essay_id = essays.id").
	LeftJoin("users ON essays.attributed_to_id = users.id")

func (h *EssayH) Perms() models.EssayPerms {
	return h.essayPerms
}
func (h EssayH) ReadView(ctx context.Context) (*models.EssayView, error) {
	if !h.essayPerms.Read {
		return nil, ErrPermDenied
	}
	sql, args, _ := selectEssayWithJoins.
		Where(sq.Eq{"essays.id": h.id}).
		GroupBy("essays.id", "essay_replies.from_id", "users.name").
		ToSql()

	var essay models.EssayView
	err := pgxscan.Get(ctx, h.sharedDB, &essay, sql, args...)
	if err != nil {
		return nil, err
	}
	return &essay, nil
}
func (h EssayH) ID() int {
	return h.id
}
func (h EssayH) CreateReport(ctx context.Context, rep models.Report, userH UserH) error {
	if !h.essayPerms.Read {
		return ErrPermDenied
	}
	if rep.EssayID != h.id || rep.FromUserID != userH.id {
		return ErrPermDenied
	}
	sql, args, _ := psql.
		Insert("reports").
		Columns("essay_id", "from_user_id", "description").
		Values(h.id, userH.id, rep.Description).
		Suffix("RETURNING id").
		ToSql()

	row := h.sharedDB.QueryRow(ctx, sql, args...)
	err := row.Scan(&rep.ID)
	if err != nil {
		return err
	}
	return nil
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
	if err != nil {
		return err
	}

	// Send notification
	if vote == models.VoteTypeUpvote {
		essay, err := h.ReadView(ctx)
		if err != nil {
			return err
		}
		if uH.id == essay.AttributedToID {
			// Don't notify self
			return nil
		}
		url, err := url.Parse(fmt.Sprintf("/s/%s/%d", essay.PostedIn, h.id))
		if err != nil {
			return err
		}
		return sendNotification(ctx, h.sharedDB, models.Notification{
			UserID:    essay.AttributedToID,
			Title:     essay.AttributedToName,
			Text:      "Upvoted your essay",
			NotifType: models.NotifTypeUpvote,
			ActionURL: *url,
		})
	}
	return nil
}
func (h EssayH) DeleteEssay(ctx context.Context) error {
	if !h.essayPerms.DeleteEssay {
		return ErrPermDenied
	}
	return h.deleteEssay(ctx)
}
func (h EssayH) ListQuestions(ctx context.Context) ([]models.Question, error) {
	sql, args, _ := psql.Select("text").From("questions").Where(sq.Eq{
		"essay_id": h.id,
	}).ToSql()
	questions := []models.Question{}
	err := pgxscan.Select(ctx, h.sharedDB, questions, sql, args...)
	if err != nil {
		return nil, err
	}
	return questions, nil
}
func (h EssayH) ListAnswers(ctx context.Context, questionID int) ([]models.Answer, error) {
	sql, args, _ := psql.Select("text", "correct").From("answers").Where(sq.Eq{
		"question_id": h.id,
	}).ToSql()
	answer := []models.Answer{}
	err := pgxscan.Select(ctx, h.sharedDB, answer, sql, args...)
	if err != nil {
		return nil, err
	}
	return answer, nil
}
func (h EssayH) GetUserDid(ctx context.Context, userH UserH) (*models.EssayUserDid, error) {
	sql, args, _ := psql.
		Select("vote_type AS vote").
		From("votes").
		Where(sq.Eq{"user_id": userH.id, "essay_id": h.id}).
		ToSql()

	did := &models.EssayUserDid{}
	err := pgxscan.Get(ctx, h.sharedDB, did, sql, args...)
	if pgxscan.NotFound(err) {
		return did, nil
	} else if err != nil {
		return nil, err
	}

	return did, nil
}
func (h EssayH) deleteEssay(ctx context.Context) error {
	sql, args, _ := psql.Delete("essays").Where(sq.Eq{"id": h.id}).ToSql()
	_, err := h.sharedDB.Exec(ctx, sql, args...)
	return err
}
