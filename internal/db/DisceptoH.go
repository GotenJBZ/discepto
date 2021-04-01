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

func (sdb *SharedDB) GetDisceptoH(ctx context.Context, uH *UserH) (*DisceptoH, error) {
	globalPerms := &models.GlobalPerms{}
	if uH != nil {
		var err error
		globalPerms, err = getGlobalUserPerms(ctx, sdb.db, uH.id)
		if err != nil {
			return nil, err
		}
	}
	return &DisceptoH{globalPerms: *globalPerms, sharedDB: sdb.db}, nil
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
func (h DisceptoH) CreateGlobalRole(ctx context.Context, globalPerms models.GlobalPerms, role string) error {
	if !h.globalPerms.ManageRole || h.globalPerms.And(globalPerms) != globalPerms {
		return ErrPermDenied
	}
	err := execTx(ctx, h.sharedDB, func(ctx context.Context, db DBTX) error {
		globalPermsID, err := createGlobalPerms(ctx, db, globalPerms)
		if err != nil {
			return err
		}
		return createGlobalRole(ctx, db, globalPermsID, role, false)
	})
	return err
}
func (h DisceptoH) AssignGlobalRole(ctx context.Context, byUser UserH, toUser UserH, role string, preset bool) error {
	if !h.globalPerms.ManageRole || !byUser.perms.Read || !toUser.perms.Read {
		return ErrPermDenied
	}
	newRolePerms, err := getGlobalRolePerms(ctx, h.sharedDB, role, preset)
	if err != nil {
		return err
	}
	if newRolePerms.And(h.globalPerms) != *newRolePerms {
		return ErrPermDenied
	}
	return assignGlobalRole(ctx, h.sharedDB, &byUser.id, toUser.id, role, preset)
}
func (h *DisceptoH) createSubdiscepto(ctx context.Context, uH UserH, subd *models.Subdiscepto) (*SubdisceptoH, error) {
	firstUserID := uH.id
	err := execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		err := createSubdiscepto(ctx, tx, *subd)
		if err != nil {
			return err
		}

		// Create a "common" role, added to every user of the subdiscepto
		subPerms := models.SubPerms{
			Read:              true,
			CreateEssay:       true,
			DeleteEssay:       false,
			BanUser:           false,
			ChangeRanking:     false,
			DeleteSubdiscepto: false,
			ManageRole:        false,
		}
		commonSubPermsID, err := createSubPerms(ctx, tx, subPerms)
		if err != nil {
			return err
		}
		err = createSubRole(ctx, tx, commonSubPermsID, subd.Name, "common", false)
		if err != nil {
			return err
		}

		// Insert first user of subdiscepto
		err = addMember(ctx, tx, subd.Name, firstUserID)
		if err != nil {
			return err
		}
		// Add "common" role
		err = assignSubRole(ctx, tx, subd.Name, nil, firstUserID, "common", false)
		if err != nil {
			return err
		}

		err = assignSubRole(ctx, tx, subd.Name, nil, firstUserID, "admin", true)
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
