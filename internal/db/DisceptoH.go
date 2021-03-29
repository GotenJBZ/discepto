package db

import (
	"context"
	"regexp"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type DisceptoH struct {
	sharedDB    DBTX
	globalPerms models.GlobalPerms
}

func (sdb *SharedDB) GetDisceptoH(ctx context.Context, uH *UserH) DisceptoH {
	perms := getGlobalPerms(ctx, sdb.db, uH)
	return DisceptoH{globalPerms: perms, sharedDB: sdb.db}
}

func (h *DisceptoH) ListUsers(ctx context.Context) ([]models.User, error) {
	// TODO: Is the list of users public? I guess not?
	var users []models.User
	err := pgxscan.Select(ctx, h.sharedDB, &users, "SELECT id, name, email FROM users")
	return users, err
}

func (h *DisceptoH) CreateSubdiscepto(ctx context.Context, uH UserH, subd *models.Subdiscepto) (*SubdisceptoH, error) {
	if !h.globalPerms.CreateSubdiscepto {
		return nil, ErrPermDenied
	}
	r := regexp.MustCompile("^\\w+$")
	if !r.Match([]byte(subd.Name)) {
		return nil, ErrInvalidFormat
	}
	return h.createSubdiscepto(ctx, uH, subd)
}
func (h *DisceptoH) createSubdiscepto(ctx context.Context, uH UserH, subd *models.Subdiscepto) (*SubdisceptoH, error) {
	firstUserID := uH.id
	err := execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		// Insert subdiscepto
		sql, args, _ := psql.
			Insert("subdisceptos").
			Columns("name", "description", "min_length", "questions_required", "nsfw", "public").
			Values(subd.Name, subd.Description, subd.MinLength, subd.QuestionsRequired, subd.Nsfw, subd.Public).
			ToSql()
		_, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}

		// Create local "common" role (added to every user)
		// Create permissions used by custom "common" role
		sql, args, _ = psql.
			Insert("sub_perms").
			Columns(
				"create_essay",
				"delete_essay",
				"ban_user",
				"change_ranking",
				"delete_subdiscepto",
				"assign_roles",
			).
			Values(true, false, false, false, false, false).
			Suffix("RETURNING id").
			ToSql()

		var subPermsID int
		row := tx.QueryRow(ctx, sql, args...)
		err = row.Scan(&subPermsID)
		if err != nil {
			return err
		}

		// Insert "common" role
		sql, args, _ = psql.
			Insert("sub_roles").
			Columns("subdiscepto", "name", "sub_perms_id", "preset").
			Values(subd.Name, "common", subPermsID, false).
			ToSql()
		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}

		// Insert first user of subdiscepto
		sql, args, _ = psql.
			Insert("subdiscepto_users").
			Columns("subdiscepto", "user_id").
			Values(subd.Name, firstUserID).
			ToSql()
		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}

		err = assignNamedSubRole(ctx, tx, firstUserID, subd.Name, "common", false)
		if err != nil {
			return err
		}

		err = assignNamedSubRole(ctx, tx, firstUserID, subd.Name, "admin", true)
		return err
	})
	if err != nil {
		return nil, err
	}
	subH := &SubdisceptoH{h.sharedDB, subd.Name, models.SubPermsOwner}
	return subH, nil
}
func (h *DisceptoH) DeleteReport(ctx context.Context, report *models.Report) error {
	// TODO: What kind of permission should one have to view reports?
	sql, args, _ := psql.
		Delete("reports").
		Where(sq.Eq{"id": report.ID}).
		ToSql()

	_, err := h.sharedDB.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
