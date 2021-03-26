package db

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type SubdisceptoH struct {
	sharedDB    DBTX
	subdiscepto string
	subPerms    models.SubPerms
}

func (sdb *SharedDB) GetSubdisceptoH(subdiscepto string, uH *UserH) (*SubdisceptoH, error) {
	var subPerms *models.SubPerms
	var err error
	if uH != nil {
		// First, try getting user's permissions
		subPerms, _ = getSubPerms(sdb.db, subdiscepto, *uH)
	}
	if subPerms == nil {
		// Check if the subdiscepto is publicly readable
		if read := isSubPublic(sdb.db, subdiscepto); read {
			subPerms = &models.SubPerms{Read: true}
		} else {
			return nil, ErrPermDenied
		}
	}

	h := &SubdisceptoH{sdb.db, subdiscepto, *subPerms}
	return h, err
}
func (h SubdisceptoH) Read() (*models.Subdiscepto, error) {
	if !h.subPerms.Read {
		return nil, ErrPermDenied
	}
	return h.read()
}
func (h SubdisceptoH) Delete() error {
	if !(h.subPerms == models.SubPermsOwner) {
		return ErrPermDenied
	}
	return h.deleteSubdiscepto()
}
func (h SubdisceptoH) CreateEssay(e *models.Essay) (*EssayH, error) {
	if !h.subPerms.CreateEssay || e.InReplyTo.Valid {
		return nil, ErrPermDenied
	}
	var essay *EssayH
	return essay, execTx(context.Background(), h.sharedDB, func(ctx context.Context, tx DBTX) error {
		var err error
		essay, err = h.createEssay(tx, e)
		return err
	})
}
func (h SubdisceptoH) CreateEssayReply(e *models.Essay, pH EssayH) (*EssayH, error) {
	if !h.subPerms.CreateEssay || e.InReplyTo.Int32 != int32(pH.id) {
		return nil, ErrPermDenied
	}
	var essay *EssayH
	return essay, execTx(context.Background(), h.sharedDB, func(ctx context.Context, tx DBTX) error {
		var err error
		essay, err = h.createEssay(tx, e)
		if err != nil {
			return err
		}
		err = createReply(ctx, tx, e)
		return err
	})
}
func createReply(ctx context.Context, db DBTX, e *models.Essay) error {
	_, err := db.Exec(ctx,
		"INSERT INTO essay_replies (from_id, to_id, reply_type) VALUES ($1, $2, $3)",
		e.ID, e.InReplyTo, e.ReplyType)
	return err
}
func (h SubdisceptoH) GetEssayH(id int, uH UserH) (*EssayH, error) {
	if !h.subPerms.Read {
		return nil, ErrPermDenied
	}
	return h.getEssayH(id, uH)
}
func (h SubdisceptoH) getEssayH(id int, uH UserH) (*EssayH, error) {
	// Check if essay is inside this subdiscepto
	sql, args, _ := psql.
		Select("1").
		From("essays").
		Where(sq.Eq{"posted_in": h.subdiscepto, "id": id}).
		ToSql()

	row := h.sharedDB.QueryRow(context.Background(), sql, args...)
	var b int
	err := row.Scan(&b)

	if err != nil {
		return nil, err
	}

	// Check if user owns the essay
	isOwner := isEssayOwner(h.sharedDB, id, uH.id)

	// Finally assign capabilities
	essayPerms := models.EssayPerms{
		DeleteEssay:   h.subPerms.DeleteEssay || isOwner,
		ChangeRanking: h.subPerms.ChangeRanking,
	}
	e := &EssayH{h.sharedDB, id, essayPerms}
	return e, nil
}
func (h SubdisceptoH) ListEssays() ([]*models.Essay, error) {
	if !h.subPerms.Read {
		return nil, ErrPermDenied
	}
	return h.listEssays()
}
func (h SubdisceptoH) ListReplies(e EssayH, replyType string) ([]*models.Essay, error) {
	if !h.subPerms.Read {
		return nil, ErrPermDenied
	}
	return h.listReplies(e, replyType)
}

func (h SubdisceptoH) createEssay(tx DBTX, essay *models.Essay) (*EssayH, error) {
	clen := len(essay.Content)
	if clen > LimitMaxContentLen || clen < LimitMinContentLen {
		return nil, ErrBadContentLen
	}

	// Insert essay
	sql, args, _ := psql.
		Insert("essays").
		Columns(
			"thesis",
			"content",
			"attributed_to_id",
			"published",
			"posted_in",
		).
		Suffix("RETURNING id").
		Values(
			essay.Thesis,
			essay.Content,
			essay.AttributedToID,
			essay.Published,
			h.subdiscepto,
		).
		ToSql()

	row := tx.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&essay.ID)
	if err != nil {
		return nil, err
	}
	err = insertTags(context.Background(), tx, essay)
	if err != nil {
		return nil, err
	}
	essayPerms := models.EssayPerms{
		Read:          true,
		DeleteEssay:   true,
		ChangeRanking: false,
	}
	return &EssayH{h.sharedDB, essay.ID, essayPerms}, err
}
func insertTags(ctx context.Context, db DBTX, essay *models.Essay) error {
	// Insert essay tags
	if len(essay.Tags) > LimitMaxTags {
		return ErrTooManyTags
	}

	// Track and skip duplicate tags
	duplicate := make(map[string]bool)

	insertCols := psql.
		Insert("essay_tags").
		Columns("essay_id", "tag")

	for _, tag := range essay.Tags {
		if duplicate[tag] {
			continue
		}
		duplicate[tag] = true

		sql, args, _ := insertCols.
			Values(essay.ID, tag).
			ToSql()

		_, err := db.Exec(ctx,
			sql, args...)
		if err != nil {
			return fmt.Errorf("Error inserting essay_tag in db: %w", err)
		}
	}
	return nil
}
func (h SubdisceptoH) read() (*models.Subdiscepto, error) {
	var sub models.Subdiscepto
	sql, args, _ := psql.
		Select("*").
		From("subdisceptos").
		Where(sq.Eq{"name": h.subdiscepto}).
		ToSql()
	err := pgxscan.Get(context.Background(), h.sharedDB, &sub, sql, args...)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
func (h SubdisceptoH) listEssays() ([]*models.Essay, error) {
	var essays []*models.Essay

	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{"posted_in": h.subdiscepto}).
		ToSql()

	err := pgxscan.Select(context.Background(), h.sharedDB, &essays, sql, args...)
	return essays, err
}
func (h SubdisceptoH) listReplies(e EssayH, replyType string) (essays []*models.Essay, err error) {
	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{
			"in_reply_to": e.id,
			"posted_in":   h.subdiscepto,
			"reply_type":  replyType,
		}).
		ToSql()

	err = pgxscan.Select(context.Background(), h.sharedDB, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
}
func (h SubdisceptoH) deleteSubdiscepto() error {
	sql, args, _ := psql.
		Delete("subdisceptos").
		Where(sq.Eq{"name": h.subdiscepto}).
		ToSql()

	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}

func isSubPublic(db DBTX, subdiscepto string) bool {
	sql, args, _ := psql.
		Select("1").
		From("subdisceptos").
		Where(sq.Eq{"name": subdiscepto, "public": true}).
		ToSql()

	row := db.QueryRow(context.Background(), sql, args...)
	var dumb int
	err := row.Scan(&dumb)
	if err != nil {
		return false
	}

	return true
}
func (h *SubdisceptoH) addMember(uH UserH) error {
	return execTx(context.Background(), h.sharedDB, func(ctx context.Context, tx DBTX) error {
		sql, args, _ := psql.
			Insert("subdiscepto_users").
			Columns("subdiscepto", "user_id").
			Values(h.subdiscepto, uH.id).
			ToSql()

		_, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}

		// Add "common" role
		return assignNamedSubRole(tx, uH.id, h.subdiscepto, "common", false)
	})
}
func (h *SubdisceptoH) removeMember(uH UserH) error {
	sql, args, _ := psql.
		Delete("subdiscepto_users").
		Where(sq.Eq{"subdiscepto": h.subdiscepto, "user_id": uH.id}).
		ToSql()

	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
