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
	subdiscepto string
	userH       *UserH
	subPerms    models.SubPerms
	sharedDB    *pgxpool.Pool
}

func (sdb *SharedDB) GetSubdisceptoH(subdiscepto string, userHandle *UserH) (SubdisceptoH, error) {
	var userHCopy UserH
	if userHandle != nil {
		userHCopy = *userHandle
	}
	h := SubdisceptoH{subdiscepto: subdiscepto, sharedDB: sdb.db, userH: &userHCopy}
	if !h.canRead() {
		return h, ErrPermDenied
	}
	subPerms, err := h.getSubPerms()
	h.subPerms = subPerms
	return h, err
}
func (h SubdisceptoH) Read() (*models.Subdiscepto, error) {
	return h.read()
}
func (h SubdisceptoH) Delete() error {
	if !h.isOwner() {
		return ErrPermDenied
	}
	return h.deleteSubdiscepto()
}
func (h SubdisceptoH) CreateEssay(e *models.Essay) (*EssayH, error) {
	if !h.subPerms.CreateEssay {
		return nil, ErrPermDenied
	}
	// If the user is trying to reply to some essay, we must have
	// an handle to the parent essay (to prevent a user replying to
	// random essays he doesn't have access to)
	parentID := int(e.InReplyTo.Int32)
	if e.InReplyTo.Valid {
		essayH, err := h.GetEssayH(parentID)
		if err != nil || essayH.id != parentID {
			return nil, ErrPermDenied
		}
	}
	return h.createEssay(e)
}
func (h SubdisceptoH) GetEssayH(id int) (EssayH, error) {
	e := EssayH{
		id:       id,
		sharedDB: h.sharedDB,
		userH:    h.userH,
	}

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
		return e, err
	}

	// Check if user owns the essay
	var isOwner bool
	if h.userH != nil {
		isOwner = e.isOwner()
	}

	// Finally assign capabilities
	e.essayPerms = models.EssayPerms{
		DeleteEssay:   h.subPerms.DeleteEssay || isOwner,
		ChangeRanking: h.subPerms.ChangeRanking,
	}
	return e, nil
}
func (h SubdisceptoH) ListEssays() ([]*models.Essay, error) {
	return h.listEssays()
}
func (h SubdisceptoH) ListReplies(e *EssayH, replyType string) ([]*models.Essay, error) {
	return h.listReplies(e, replyType)
}

func (h SubdisceptoH) createEssay(essay *models.Essay) (*EssayH, error) {
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
			"in_reply_to",
			"reply_type",
		).
		Suffix("RETURNING id").
		Values(
			essay.Thesis,
			essay.Content,
			essay.AttributedToID,
			essay.Published,
			h.subdiscepto,
			essay.InReplyTo,
			essay.ReplyType,
		).
		ToSql()

	err := execTx(context.Background(), *h.sharedDB, func(ctx context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(ctx, sql, args...)
		err := row.Scan(&essay.ID)
		if err != nil {
			return err
		}
		err = insertTags(ctx, tx, essay)
		return err
	})
	return &EssayH{
		id:    essay.ID,
		userH: h.userH,
		essayPerms: models.EssayPerms{
			DeleteEssay:   true,
			ChangeRanking: false,
		},
		sharedDB: h.sharedDB,
	}, err
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
func (h SubdisceptoH) listReplies(e *EssayH, replyType string) (essays []*models.Essay, err error) {
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
// The user may have multiple roles:
// - preset (already defined when Discepto is installed)
// - custom (defined by community admins at runtime)
// The user may have a global role, granting him permissions inside every subdiscepto.
// That means we have 3 tables to check:
// - user_preset_sub_roles
// - user_custom_sub_roles
// - user_global_roles
// We simply fetch all the roles assigned to a user, get the corresponding permission row
// and UNION the results. Then we use the aggregate function "bool_or" to sum
// every premission. The result is 1 row with the correct permissions.
func (h SubdisceptoH) getSubPerms() (perms models.SubPerms, err error) {
	// TODO: Check global roles

	queryPresetSubRoles := sq.Select("sub_perms_id").
		From("user_preset_sub_roles").
		Join("preset_sub_roles ON user_preset_sub_roles.role_name = preset_sub_roles.name").
		Where(sq.Eq{"user_preset_sub_roles.subdiscepto": h.subdiscepto, "user_id": h.userH.userID})

	queryCustomSubPerms := sq.Select("sub_perms_id").
		From("user_custom_sub_roles").
		Join("custom_sub_roles ON user_custom_sub_roles.role_name = custom_sub_roles.name AND user_custom_sub_roles.subdiscepto = custom_sub_roles.subdiscepto").
		Where(sq.Eq{"custom_sub_roles.subdiscepto": h.subdiscepto, "user_id": h.userH.userID})

	queryAllSubPerms := queryPresetSubRoles.Suffix("UNION").SuffixExpr(queryCustomSubPerms)

	sql, args, _ := psql.
		Select(
			bool_or("create_essay"),
			bool_or("delete_essay"),
			bool_or("ban_user"),
			bool_or("change_ranking"),
			bool_or("delete_subdiscepto"),
			bool_or("add_mod"),
		).
		FromSelect(queryAllSubPerms, "user_perms_ids").
		Join("sub_perms ON sub_perms.id = user_perms_ids.sub_perms_id").
		PlaceholderFormat(sq.Dollar).
		ToSql()

	row := h.sharedDB.QueryRow(context.Background(), sql, args...)
	err = row.Scan(
		&perms.CreateEssay,
		&perms.DeleteEssay,
		&perms.BanUser,
	        &perms.ChangeRanking,
	        &perms.DeleteSubdiscepto,
	        &perms.AddMod,
	)
	if err == pgx.ErrNoRows {
		return perms, nil
	}
	return perms, err
}
func (h SubdisceptoH) canRead() bool {
	queryPrivate := sq.
		Select("private").
		From("subdisceptos").
		Where(sq.Eq{"name": h.subdiscepto, "private": true})

	sql, args, _ := sq.Select("private", "is_member").
		FromSelect(queryPrivate, "query_private").
		Join("subdiscepto_users").
		Where(sq.Eq{"subdiscepto": h.subdiscepto, "user_id": h.userH.userID}).
		ToSql()

	row := h.sharedDB.QueryRow(context.Background(), sql, args...)
	private := false
	isMember := false
	err := row.Scan(&private, &isMember)
	if err == pgx.ErrNoRows {
		// If there are no rows, that means the subdiscepto is public.
		// If the query failed because the subdiscepto doesn't exist anymore
		// data can't be read anyway
		return true
	}

	return !private || isMember
}
func (h SubdisceptoH) isOwner() bool {
	return h.subPerms == models.SubPermsOwner
}
func (h *SubdisceptoH) Join() error {
	return execTx(context.Background(), *h.sharedDB, func(ctx context.Context, tx pgx.Tx) error {
		sql, args, _ := psql.
			Insert("subdiscepto_users").
			Columns("subdiscepto", "user_id").
			Values(h.subdiscepto, h.userH.userID).
			ToSql()

		_, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}

		// Add "common" role
		sql, args, _ = psql.
			Insert("user_custom_sub_roles").
			Columns("subdiscepto", "user_id", "role_name").
			Values(h.subdiscepto, h.userH.userID, "common").
			ToSql()

		_, err = tx.Exec(ctx, sql, args...)
		return err
	})
}
func (h *SubdisceptoH) Leave() error {
	sql, args, _ := psql.
		Delete("subdiscepto_users").
		Where(sq.Eq{"subdiscepto": h.subdiscepto, "user_id": h.userH.userID}).
		ToSql()

	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
