package db

import (
	"context"
	"regexp"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type DisceptoH struct {
	sharedDB    *pgxpool.Pool
	globalPerms models.GlobalPerms
	userH       *UserH
}

func (sdb *SharedDB) GetDisceptoH(userHandle *UserH) DisceptoH {
	userHCopy := UserH{}
	if userHandle != nil {
		userHCopy = *userHandle
	}
	perms := sdb.getGlobalPerms(&userHCopy)
	return DisceptoH{globalPerms: perms, userH: &userHCopy, sharedDB: sdb.db}
}

func (sdb *SharedDB) getGlobalPerms(userH *UserH) models.GlobalPerms {
	perms := models.GlobalPerms{}
	if userH != nil {
		sql, args, _ := psql.Select(
			bool_or("login"),
			bool_or("create_subdiscepto"),
			bool_or("ban_user_globally"),
			bool_or("delete_user"),
			bool_or("add_admin"),
		).
			From("global_perms").
			Join("global_roles ON global_roles.global_perms_id = global_perms.id").
			Join("user_global_roles ON user_global_roles.role_name = global_roles.name").
			Where(sq.Eq{"user_id": userH.userID}).
			ToSql()

		pgxscan.Get(context.Background(), sdb.db, &perms, sql, args...)
	}
	return perms
}

func (h *DisceptoH) ListUsers() ([]models.User, error) {
	var users []models.User
	err := pgxscan.Select(context.Background(), h.sharedDB, &users, "SELECT id, name, email FROM users")
	return users, err
}

func (h *DisceptoH) CreateSubdiscepto(subd *models.Subdiscepto) (SubdisceptoH, error) {
	subH := SubdisceptoH{}
	if !h.globalPerms.CreateSubdiscepto {
		return subH, ErrPermDenied
	}
	return h.createSubdiscepto(subd)
}
func (h *DisceptoH) createSubdiscepto(subd *models.Subdiscepto) (SubdisceptoH, error) {
	subH := SubdisceptoH{userH: h.userH}
	r := regexp.MustCompile("^\\w+$")
	if !r.Match([]byte(subd.Name)) {
		return subH, ErrInvalidFormat
	}

	firstUserID := h.userH.userID
	err := execTx(context.Background(), *h.sharedDB, func(ctx context.Context, tx pgx.Tx) error {
		// Insert subdiscepto
		sql, args, _ := psql.
			Insert("subdisceptos").
			Columns("name", "description", "min_length", "questions_required", "nsfw").
			Values(subd.Name, subd.Description, subd.MinLength, subd.QuestionsRequired, subd.Nsfw).
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
				"add_mod",
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

		// Add "common" role to first user
		sql, args, _ = psql.
			Insert("custom_sub_roles").
			Columns("subdiscepto", "name", "sub_perms_id").
			Values(subd.Name, "common", subPermsID).
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

		// Assign admin role to first user
		sql, args, _ = psql.
			Insert("user_preset_sub_roles").
			Columns("subdiscepto", "user_id", "role_name").
			Values(subd.Name, firstUserID, "admin").
			ToSql()

		_, err = tx.Exec(ctx, sql, args...)
		return err
	})
	if err != nil {
		return subH, err
	}
	subH = SubdisceptoH{
		subdiscepto: subd.Name,
		subPerms:    models.SubPermsOwner,
		sharedDB:    h.sharedDB,
		userH:       h.userH,
	}
	return subH, nil
}
func (h *DisceptoH) DeleteReport(report *models.Report) error {
	sql, args, _ := psql.
		Delete("reports").
		Where(sq.Eq{"id": report.ID}).
		ToSql()

	_, err := h.sharedDB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
