package db

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type SubdisceptoH struct {
	sharedDB    *pgxpool.Pool
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
		if read := isPublic(sdb.db, subdiscepto, uH.id); read {
			subPerms = &models.SubPerms{}
		} else {
			return nil, ErrPermDenied
		}
	}

	fmt.Println(subPerms)
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
	return essay, execTx(context.Background(), *h.sharedDB, func(ctx context.Context, tx pgx.Tx) error {
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
	return essay, execTx(context.Background(), *h.sharedDB, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		essay, err = h.createEssay(tx, e)
		if err != nil {
			return err
		}
		err = createReply(ctx, tx, e)
		return err
	})
}
func createReply(ctx context.Context, tx pgx.Tx, e *models.Essay) error {
	_, err := tx.Exec(ctx,
		"INSERT INTO essay_replies (from_id, to_id, reply_type) VALUES ($1, $2, $3)",
		e.ID, e.InReplyTo, e.ReplyType)
	return err
}
func (h SubdisceptoH) GetEssayH(id int, uH UserH) (*EssayH, error) {
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
	return h.listEssays()
}
func (h SubdisceptoH) ListReplies(e EssayH, replyType string) ([]*models.Essay, error) {
	return h.listReplies(e, replyType)
}

func (h SubdisceptoH) createEssay(tx pgx.Tx, essay *models.Essay) (*EssayH, error) {
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
		DeleteEssay:   true,
		ChangeRanking: false,
	}
	return &EssayH{h.sharedDB, essay.ID, essayPerms}, err
}
func insertTags(ctx context.Context, tx pgx.Tx, essay *models.Essay) error {
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

		_, err := tx.Exec(ctx,
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

// Returns the permissions corresponding to a user inside a subdiscepto.
// The user may have multiple roles and may also have a global role,
// granting him permissions inside every subdiscepto.
// We simply fetch all the roles assigned to a user, get the corresponding permission row
// and UNION the results. Then we use the aggregate function "bool_or" to sum
// every premission. The result is 1 row with the correct permissions.
func getSubPerms(db *pgxpool.Pool, subdiscepto string, uH UserH) (perms *models.SubPerms, err error) {
	// TODO: Check global roles

	querySubRolesPermsID := sq.Select("sub_perms_id").
		From("user_sub_roles").
		Where(sq.Eq{"subdiscepto": subdiscepto, "user_id": uH.id})

	sql, args, _ := psql.
		Select(
			bool_or("create_essay"),
			bool_or("delete_essay"),
			bool_or("ban_user"),
			bool_or("change_ranking"),
			bool_or("delete_subdiscepto"),
			bool_or("add_mod"),
		).
		FromSelect(querySubRolesPermsID, "user_perms_ids").
		Join("sub_perms ON sub_perms.id = user_perms_ids.sub_perms_id").
		PlaceholderFormat(sq.Dollar).
		Having("COUNT(*) > 0").
		ToSql()

	row := db.QueryRow(context.Background(), sql, args...)
	perms = &models.SubPerms{}
	err = row.Scan(
		&perms.CreateEssay,
		&perms.DeleteEssay,
		&perms.BanUser,
		&perms.ChangeRanking,
		&perms.DeleteSubdiscepto,
		&perms.AddMod,
	)
	perms.Read = true // FIXME: INSECURE!!!!!
	perms.EssayPerms.Read = true
	if err != nil {
		return nil, err
	}
	return perms, nil
}
func isPublic(db *pgxpool.Pool, subdiscepto string, userID int) bool {
	sql, args, _ := sq.
		Select("1").
		From("subdisceptos").
		Where(sq.Eq{"name": subdiscepto, "private": false}).
		ToSql()

	row := db.QueryRow(context.Background(), sql, args...)
	var dumb int
	err := row.Scan(&dumb)
	if err == pgx.ErrNoRows {
		return false
	}

	return true
}
func (h *SubdisceptoH) addMember(uH UserH) error {
	return execTx(context.Background(), *h.sharedDB, func(ctx context.Context, tx pgx.Tx) error {
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
