package db

import (
	"context"
	"fmt"
	"regexp"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type DisceptoH struct {
	sharedDB    DBTX
	globalPerms models.GlobalPerms
}

func (sdb *SharedDB) GetDisceptoH(ctx context.Context, uH *UserH) (*DisceptoH, error) {
	globalPerms := models.GlobalPerms{}
	if uH != nil {
		perms, err := getUserPerms(ctx, sdb.db, "discepto", uH.id)
		if err != nil {
			return nil, err
		}
		globalPerms = models.GlobalPermsFromMap(perms)
		if err != nil {
			return nil, err
		}
	}
	return &DisceptoH{globalPerms: globalPerms, sharedDB: sdb.db}, nil
}

func (h *DisceptoH) Perms() models.GlobalPerms {
	return h.globalPerms
}
func (h *DisceptoH) ListRoles(ctx context.Context) ([]models.Role, error) {
	if !h.globalPerms.ManageRole {
		return nil, ErrPermDenied
	}
	return listRoles(ctx, h.sharedDB, "discepto")
}
func (h *DisceptoH) ListMembers(ctx context.Context) ([]models.Member, error) {
	if !h.globalPerms.Login {
		return nil, ErrPermDenied
	}

	sqlquery, args, _ := psql.
		Select("users.id AS user_id", "users.name").
		From("users").
		OrderBy("users.id").
		ToSql()

	members := []models.Member{}
	err := pgxscan.Select(ctx, h.sharedDB, &members, sqlquery, args...)
	if err != nil {
		return nil, err
	}

	for i := range members {
		members[i].Roles, err = listUserRoles(ctx, h.sharedDB, members[i].UserID, "discepto")
	}

	return members, nil
}
func (h *DisceptoH) ReadPublicUser(ctx context.Context, userID int) (*models.User, error) {
	if !h.globalPerms.Login {
		return nil, ErrPermDenied
	}
	user := &models.User{}
	sql, args, _ := psql.
		Select("users.name", "users.id").
		From("users").
		Where(sq.Eq{"id": userID}).
		ToSql()

	err := pgxscan.Get(
		ctx,
		h.sharedDB, user,
		sql, args...)

	if err != nil {
		return nil, err
	}
	return user, nil
}

func (h *DisceptoH) CreateSubdiscepto(ctx context.Context, uH UserH, subd models.Subdiscepto) (*SubdisceptoH, error) {
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
	_, err := createRole(ctx, h.sharedDB, "discepto", role, false, globalPerms.ToBoolMap())
	return err
}
func (h *DisceptoH) AssignRole(ctx context.Context, byUser UserH, toUser int, roleH RoleH) error {
	if !h.globalPerms.ManageRole ||
		!byUser.perms.Read ||
		!roleH.rolePerms.ManageRole ||
		!(roleH.domain == "discepto") {
		return ErrPermDenied
	}
	newRolePerms, err := roleH.ListActivePerms(ctx)
	if err != nil {
		return err
	}
	globalPerms := models.GlobalPermsFromMap(newRolePerms)
	if globalPerms.And(h.globalPerms) != globalPerms {
		return ErrPermDenied
	}
	return assignRole(ctx, h.sharedDB, toUser, roleH.id)
}
func (h *DisceptoH) UnassignRole(ctx context.Context, toUser int, roleH RoleH) error {
	if !h.globalPerms.ManageRole || !roleH.rolePerms.ManageRole || !(roleH.domain == "discepto") {
		return ErrPermDenied
	}
	newRolePerms, err := roleH.ListActivePerms(ctx)
	if err != nil {
		return err
	}
	globalPerms := models.GlobalPermsFromMap(newRolePerms)
	if globalPerms.And(h.globalPerms) != globalPerms {
		return ErrPermDenied
	}
	return unassignRole(ctx, h.sharedDB, toUser, roleH.id)
}
func (h *DisceptoH) ListAvailablePerms() map[string]bool {
	return models.GlobalPerms{}.ToBoolMap()
}
func (h *DisceptoH) createSubdiscepto(ctx context.Context, uH UserH, subd models.Subdiscepto) (*SubdisceptoH, error) {
	firstUserID := uH.id
	err := execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		err := createSubdiscepto(ctx, tx, subd)
		if err != nil {
			return err
		}

		// Create a "common" role, added to every user of the subdiscepto
		subPerms := models.SubPerms{
			ReadSubdiscepto:   true,
			CreateEssay:       true,
			UpdateSubdiscepto: false,
			DeleteEssay:       false,
			BanUser:           false,
			ChangeRanking:     false,
			DeleteSubdiscepto: false,
			ManageRole:        false,
			CommonAfterRejoin: true,
		}
		p := subPerms.ToBoolMap()
		_, err = createRole(ctx, tx, subRoleDomain(subd.Name), "common", false, p)
		if err != nil {
			return err
		}

		// Create a "common-after-rejoin" role, added to every user while away from the subdiscepto
		subPerms = models.SubPerms{
			CommonAfterRejoin: true,
		}
		p = subPerms.ToBoolMap()
		_, err = createRole(ctx, tx, subRoleDomain(subd.Name), "common-after-rejoin", false, p)
		if err != nil {
			return err
		}

		// Create an "admin" role, added to the first user
		subPerms = models.SubPerms{
			ReadSubdiscepto:   true,
			CreateEssay:       true,
			UpdateSubdiscepto: true,
			DeleteEssay:       true,
			BanUser:           true,
			ChangeRanking:     true,
			DeleteSubdiscepto: true,
			ManageRole:        true,
			ViewReport:        true,
			DeleteReport:      true,
		}
		adminRoleID, err := createRole(ctx, tx, subRoleDomain(subd.Name), "admin", true, subPerms.ToBoolMap())
		if err != nil {
			return err
		}

		// Insert first user of subdiscepto
		err = addMember(ctx, tx, subd.Name, firstUserID)
		if err != nil {
			return err
		}

		err = assignRole(ctx, tx, firstUserID, adminRoleID)
		fmt.Println(err)
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
func (h *DisceptoH) ListRecentEssaysIn(ctx context.Context, subs []string) ([]models.EssayView, error) {
	essayPreviews := []models.EssayView{}
	sql, args, _ := selectEssayWithJoins.
		Where(sq.Eq{"posted_in": subs}).
		GroupBy("essays.id", "essay_replies.from_id", "users.name").
		OrderBy("essays.id DESC").
		ToSql()

	err := pgxscan.Select(ctx, h.sharedDB, &essayPreviews, sql, args...)
	if err != nil {
		return nil, err
	}
	return essayPreviews, nil
}
func (sdb *SharedDB) ListSubdisceptos(ctx context.Context, userH *UserH) ([]models.SubdisceptoView, error) {
	var subs []models.SubdisceptoView
	var userID *int
	if userH != nil {
		userID = &userH.id
	}
	sql, args, _ := selectSubdiscepto(userID).ToSql()
	err := pgxscan.Select(ctx, sdb.db, &subs, sql, args...)
	if err != nil {
		return nil, err
	}
	return subs, nil
}
func (h *DisceptoH) SearchByTags(ctx context.Context, tags []string) ([]models.EssayView, error) {
	sql, args, _ := selectEssayWithJoins.
		Join("essay_tags ON essays.id = essay_tags.essay_id").
		Join("subdisceptos ON subdisceptos.name = essays.posted_in").
		GroupBy("essays.id", "essay_replies.from_id", "users.name").
		Where(sq.Eq{"subdisceptos.public": true, "essay_tags.tag": tags}).
		OrderBy("essays.id DESC").
		ToSql()

	fmt.Println(sql)
	rows, err := h.sharedDB.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	essays, err := scanEssays(ctx, rows, tags)
	if err != nil {
		return nil, err
	}

	return essays, nil
}
func scanEssays(ctx context.Context, rows pgx.Rows, tags []string) ([]models.EssayView, error) {
	essayMap := map[int]*models.EssayView{}
	for rows.Next() {
		tmp := &models.EssayRow{}
		err := pgxscan.ScanRow(tmp, rows)
		if err != nil {
			return nil, err
		}
		if essay, ok := essayMap[tmp.ID]; ok {
			essay.Tags = append(essay.Tags, tmp.Tag)
		} else {
			essayMap[tmp.ID] = &models.EssayView{
				ID:               tmp.ID,
				Thesis:           tmp.Thesis,
				Content:          tmp.Content,
				Published:        tmp.Published,
				PostedIn:         tmp.PostedIn,
				AttributedToID:   tmp.AttributedToID,
				AttributedToName: tmp.AttributedToName,
				Upvotes:          tmp.Upvotes,
				Downvotes:        tmp.Downvotes,
				Tags:             []string{tmp.Tag},
				Replying: models.Replying{
					InReplyTo: tmp.InReplyTo,
					ReplyType: tmp.ReplyType,
				},
			}
		}
	}
	n := rows.CommandTag().RowsAffected()
	essays := make([]models.EssayView, 0, n)
	for _, v := range essayMap {
		essays = append(essays, *v)
	}

	return essays, nil
}
func (h *DisceptoH) SearchByThesis(ctx context.Context, title string) ([]models.EssayView, error) {
	sql, args, _ := selectEssayWithJoins.
		Join("subdisceptos ON subdisceptos.name = essays.posted_in").
		Where("subdisceptos.public = true AND essays.thesis ILIKE $1", fmt.Sprintf(`%%%s%%`, title)).
		GroupBy("essays.id", "essay_replies.from_id", "users.name").
		OrderBy("essays.id DESC").
		ToSql()

	essays := []models.EssayView{}

	err := pgxscan.Select(ctx, h.sharedDB, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
}
