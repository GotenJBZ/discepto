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
	RolesH
	sharedDB     DBTX
	globalPerms  models.GlobalPerms
	notifService models.NotificationService
}

func (sdb *SharedDB) GetDisceptoH(ctx context.Context, uH *UserH) (*DisceptoH, error) {
	globalPerms := models.GlobalPerms{}
	if uH != nil {
		perms, err := getUserPerms(ctx, sdb.db, models.RoleDomainDiscepto, uH.id)
		if err != nil {
			return nil, err
		}
		globalPerms = models.GlobalPermsFromMap(perms)
		if err != nil {
			return nil, err
		}
	}
	rolesH := RolesH{
		contextPerms: globalPerms.ToBoolMap(),
		rolesPerms: struct {
			ManageRoles bool
		}{globalPerms.ManageGlobalRole},
		domain:   models.RoleDomainDiscepto,
		sharedDB: sdb.db,
	}
	notifService := NewNotificationService(sdb.db)
	return &DisceptoH{globalPerms: globalPerms, RolesH: rolesH, sharedDB: sdb.db, notifService: notifService}, nil
}

func (h *DisceptoH) Perms() models.GlobalPerms {
	return h.globalPerms
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
		members[i].Roles, err = h.ListUserRoles(ctx, members[i].UserID)
	}

	return members, nil
}
func (h *DisceptoH) ReadPublicUser(ctx context.Context, userID int) (*models.UserView, error) {
	if !h.globalPerms.Login {
		return nil, ErrPermDenied
	}
	return readPublicUser(ctx, h.sharedDB, userID)
}

func (h *DisceptoH) CreateSubdiscepto(ctx context.Context, uH UserH, sub *models.SubdisceptoReq) (*SubdisceptoH, error) {
	if !h.globalPerms.CreateSubdiscepto {
		return nil, ErrPermDenied
	}
	r := regexp.MustCompile("^\\w+$")
	if !r.Match([]byte(sub.Name)) {
		return nil, ErrInvalidFormat
	}
	return h.createSubdiscepto(ctx, uH, sub)
}
func (h *DisceptoH) ListAvailablePerms() map[string]bool {
	return models.GlobalPerms{}.ToBoolMap()
}
func (h *DisceptoH) createSubdiscepto(ctx context.Context, uH UserH, subd *models.SubdisceptoReq) (*SubdisceptoH, error) {
	firstUserID := uH.id

	// Retrieve real roledomain
	roledomain, err := createRoledomain(ctx, h.sharedDB, "subdiscepto")
	if err != nil {
		return nil, err
	}
	rawSub := &models.Subdiscepto{
		Name:              subd.Name,
		Description:       subd.Description,
		RoledomainID:      roledomain,
		MinLength:         subd.MinLength,
		QuestionsRequired: subd.QuestionsRequired,
		Nsfw:              subd.Nsfw,
		Public:            subd.Public,
	}

	// Init subH
	subH := &SubdisceptoH{
		sharedDB: h.sharedDB,
		rawSub:   rawSub,
		RolesH: RolesH{
			contextPerms: models.SubPermsOwner.ToBoolMap(),
			rolesPerms:   struct{ ManageRoles bool }{true},
			domain:       roledomain,
			sharedDB:     h.sharedDB,
		},
		subPerms:     models.SubPermsOwner,
		notifService: h.notifService,
	}
	err = execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		err := insertSubdiscepto(ctx, tx, *rawSub)
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
		_, err = createRole(ctx, tx, rawSub.RoledomainID, "common", false, p)
		if err != nil {
			return err
		}

		// Create a "common-after-rejoin" role, added to every user while away from the subdiscepto
		subPerms = models.SubPerms{
			CommonAfterRejoin: true,
		}
		p = subPerms.ToBoolMap()
		_, err = createRole(ctx, tx, rawSub.RoledomainID, "common-after-rejoin", false, p)
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
		adminRoleID, err := createRole(ctx, tx, subH.domain, "admin", true, subPerms.ToBoolMap())
		if err != nil {
			return err
		}

		// Insert first user of subdiscepto
		err = addMember(ctx, tx, rawSub, firstUserID)
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
	sql, args, _ := selectSubdiscepto(userID).Where(sq.Eq{"public": true}).ToSql()
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
func (h *DisceptoH) ListUserEssays(ctx context.Context, userID int) ([]models.EssayView, error) {
	return listUserEssays(ctx, h.sharedDB, userID)
}
func (h *DisceptoH) ListNotifs(ctx context.Context, userH *UserH) ([]models.NotifView, error) {
	if !userH.perms.Read {
		return nil, ErrPermDenied
	}
	return h.notifService.List(ctx, userH.id)
}
func (h *DisceptoH) DeleteNotif(ctx context.Context, userH *UserH, notifID int) error {
	if !userH.perms.Read {
		return ErrPermDenied
	}
	return h.notifService.Delete(ctx, userH.id, notifID)
}
func readPublicUser(ctx context.Context, db DBTX, userID int) (*models.UserView, error) {
	user := &models.UserView{}
	sql, args, _ := psql.
		Select(
			"users.name",
			"users.id",
			"users.created_at",
			"SUM(CASE votes.vote_type WHEN 'upvote' THEN 1 ELSE 0 END) AS karma",
		).
		From("users").
		LeftJoin("essays ON essays.attributed_to_id = users.id").
		LeftJoin("votes ON essays.id = votes.essay_id").
		GroupBy("users.id").
		Where(sq.Eq{"users.id": userID}).
		ToSql()

	err := pgxscan.Get(
		ctx,
		db, user,
		sql, args...)

	if err != nil {
		return nil, err
	}
	return user, nil
}
