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

func (sdb *SharedDB) GetSubdisceptoH(ctx context.Context, subdiscepto string, uH *UserH) (*SubdisceptoH, error) {
	var subPerms *models.SubPerms
	var err error
	if uH != nil {
		// First, try getting user's permissions
		subPerms, err = getSubUserPerms(ctx, sdb.db, subdiscepto, uH.id)
		if err != nil {
			return nil, err
		}
	}
	if subPerms == nil || !subPerms.ReadSubdiscepto {
		// Check if the subdiscepto is publicly readable
		public := isSubPublic(ctx, sdb.db, subdiscepto)
		if !public {
			return nil, ErrPermDenied
		}
		subPerms = &models.SubPerms{ReadSubdiscepto: public}
	}

	h := &SubdisceptoH{sdb.db, subdiscepto, *subPerms}
	return h, nil
}
func (h SubdisceptoH) Read(ctx context.Context) (*models.Subdiscepto, error) {
	if !h.subPerms.ReadSubdiscepto {
		return nil, ErrPermDenied
	}
	return h.read(ctx)
}
func (h SubdisceptoH) Delete(ctx context.Context) error {
	if h.subPerms != models.SubPermsOwner {
		return ErrPermDenied
	}
	return h.deleteSubdiscepto(ctx)
}
func (h SubdisceptoH) CreateEssay(ctx context.Context, e *models.Essay) (*EssayH, error) {
	if !h.subPerms.CreateEssay || e.InReplyTo.Valid {
		return nil, ErrPermDenied
	}
	var essay *EssayH
	return essay, execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		var err error
		essay, err = h.createEssay(ctx, tx, e)
		return err
	})
}
func (h SubdisceptoH) CreateEssayReply(ctx context.Context, e *models.Essay, pH EssayH) (*EssayH, error) {
	if !h.subPerms.CreateEssay || e.InReplyTo.Int32 != int32(pH.id) {
		return nil, ErrPermDenied
	}
	var essay *EssayH
	return essay, execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		var err error
		essay, err = h.createEssay(ctx, tx, e)
		if err != nil {
			return err
		}
		err = createReply(ctx, tx, e)
		return err
	})
}
func (h SubdisceptoH) CreateRole(ctx context.Context, subPerms models.SubPerms, role string) error {
	if !h.subPerms.ManageRole || h.subPerms.And(subPerms) != subPerms {
		return ErrPermDenied
	}
	err := execTx(ctx, h.sharedDB, func(ctx context.Context, db DBTX) error {
		subPermsID, err := createSubPerms(ctx, db, subPerms)
		if err != nil {
			return err
		}
		return createSubRole(ctx, db, subPermsID, h.subdiscepto, role, false)
	})

	return err
}
func (h SubdisceptoH) AssignRole(ctx context.Context, byUser UserH, toUser int, role string, preset bool) error {
	if !h.subPerms.ManageRole || !byUser.perms.Read {
		return ErrPermDenied
	}
	newRolePerms, err := getSubRolePerms(ctx, h.sharedDB, h.subdiscepto, role, preset)
	if err != nil {
		return err
	}
	if newRolePerms.And(h.subPerms) != *newRolePerms {
		return ErrPermDenied
	}
	return assignSubRole(ctx, h.sharedDB, h.subdiscepto, &byUser.id, toUser, role, preset)
}
func (h SubdisceptoH) AddMember(ctx context.Context, userH UserH) error {
	if !h.subPerms.ReadSubdiscepto || !userH.perms.Read {
		return ErrPermDenied
	}
	err := addMember(ctx, h.sharedDB, h.subdiscepto, userH.id)
	if err != nil {
		return err
	}
	return assignSubRole(ctx, h.sharedDB, h.subdiscepto, nil, userH.id, "common", false)
}
func (h SubdisceptoH) RemoveMember(ctx context.Context, userH UserH) error {
	// TODO: should check for specific permission to remove other users
	if !h.subPerms.ReadSubdiscepto || !userH.perms.Read {
		return ErrPermDenied
	}
	return removeMember(ctx, h.sharedDB, h.subdiscepto, userH.id)
}
func createReply(ctx context.Context, db DBTX, e *models.Essay) error {
	_, err := db.Exec(ctx,
		"INSERT INTO essay_replies (from_id, to_id, reply_type) VALUES ($1, $2, $3)",
		e.ID, e.InReplyTo, e.ReplyType)
	return err
}
func (h SubdisceptoH) GetEssayH(ctx context.Context, id int, uH UserH) (*EssayH, error) {
	if !h.subPerms.ReadSubdiscepto {
		return nil, ErrPermDenied
	}
	return h.getEssayH(ctx, id, uH)
}
func (h SubdisceptoH) getEssayH(ctx context.Context, id int, uH UserH) (*EssayH, error) {
	// Check if essay is inside this subdiscepto
	sql, args, _ := psql.
		Select("1").
		From("essays").
		Where(sq.Eq{"posted_in": h.subdiscepto, "id": id}).
		ToSql()

	row := h.sharedDB.QueryRow(ctx, sql, args...)
	var b int
	err := row.Scan(&b)

	if err != nil {
		return nil, err
	}

	// Check if user owns the essay
	isOwner := isEssayOwner(ctx, h.sharedDB, id, uH.id)

	// Finally assign capabilities
	essayPerms := models.EssayPerms{
		Read:          h.subPerms.ReadSubdiscepto || isOwner,
		DeleteEssay:   h.subPerms.DeleteEssay || isOwner,
		ChangeRanking: false, // TODO: to implement in future
	}
	e := &EssayH{h.sharedDB, id, essayPerms}
	return e, nil
}
func (h SubdisceptoH) ListEssays(ctx context.Context) ([]*models.Essay, error) {
	if !h.subPerms.ReadSubdiscepto {
		return nil, ErrPermDenied
	}
	return h.listEssays(ctx)
}
func (h SubdisceptoH) ListReplies(ctx context.Context, e EssayH, replyType string) ([]*models.Essay, error) {
	if !h.subPerms.ReadSubdiscepto {
		return nil, ErrPermDenied
	}
	return h.listReplies(ctx, e, replyType)
}

func (h SubdisceptoH) createEssay(ctx context.Context, tx DBTX, essay *models.Essay) (*EssayH, error) {
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

	row := tx.QueryRow(ctx, sql, args...)
	err := row.Scan(&essay.ID)
	if err != nil {
		return nil, err
	}
	err = insertTags(ctx, tx, essay)
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
func (h SubdisceptoH) read(ctx context.Context) (*models.Subdiscepto, error) {
	var sub models.Subdiscepto
	sql, args, _ := psql.
		Select("*").
		From("subdisceptos").
		Where(sq.Eq{"name": h.subdiscepto}).
		ToSql()
	err := pgxscan.Get(ctx, h.sharedDB, &sub, sql, args...)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
func (h SubdisceptoH) listEssays(ctx context.Context) ([]*models.Essay, error) {
	var essays []*models.Essay

	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{"posted_in": h.subdiscepto}).
		ToSql()

	err := pgxscan.Select(ctx, h.sharedDB, &essays, sql, args...)
	return essays, err
}
func (h SubdisceptoH) listReplies(ctx context.Context, e EssayH, replyType string) (essays []*models.Essay, err error) {
	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{
			"in_reply_to": e.id,
			"posted_in":   h.subdiscepto,
			"reply_type":  replyType,
		}).
		ToSql()

	err = pgxscan.Select(ctx, h.sharedDB, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
}
func (h SubdisceptoH) deleteSubdiscepto(ctx context.Context) error {
	sql, args, _ := psql.
		Delete("subdisceptos").
		Where(sq.Eq{"name": h.subdiscepto}).
		ToSql()

	_, err := h.sharedDB.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}

func isSubPublic(ctx context.Context, db DBTX, subdiscepto string) bool {
	sql, args, _ := psql.
		Select("1").
		From("subdisceptos").
		Where(sq.Eq{"name": subdiscepto, "public": true}).
		ToSql()

	row := db.QueryRow(ctx, sql, args...)
	var dumb int
	err := row.Scan(&dumb)
	if err != nil {
		return false
	}

	return true
}
func addMember(ctx context.Context, db DBTX, subdiscepto string, userID int) error {
	return execTx(ctx, db, func(ctx context.Context, tx DBTX) error {
		sql, args, _ := psql.
			Insert("subdiscepto_users").
			Columns("subdiscepto", "user_id").
			Values(subdiscepto, userID).
			ToSql()

		_, err := tx.Exec(ctx, sql, args...)
		return err
	})
}
func removeMember(ctx context.Context, db DBTX, subdiscepto string, userID int) error {
	sql, args, _ := psql.
		Delete("subdiscepto_users").
		Where(sq.Eq{"subdiscepto": subdiscepto, "user_id": userID}).
		ToSql()

	_, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func createSubdiscepto(ctx context.Context, db DBTX, sub models.Subdiscepto) error {
	// Insert subdiscepto
	sql, args, _ := psql.
		Insert("subdisceptos").
		Columns("name", "description", "min_length", "questions_required", "nsfw", "public").
		Values(sub.Name, sub.Description, sub.MinLength, sub.QuestionsRequired, sub.Nsfw, sub.Public).
		ToSql()
	_, err := db.Exec(ctx, sql, args...)
	return err
}
