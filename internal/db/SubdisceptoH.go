package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/georgysavva/scany/pgxscan"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/ranfdev/discepto/internal/models"
)

const SubRolePrefix = "subdisceptos"

func subRoleDomain(subdiscepto string) RoleDomain {
	return RoleDomain(fmt.Sprintf("%s/%s", SubRolePrefix, subdiscepto))
}

type SubdisceptoH struct {
	sharedDB DBTX
	name     string
	RolesH
	subPerms models.SubPerms
}

func (dH *DisceptoH) GetSubdisceptoH(ctx context.Context, subdiscepto string, uH *UserH) (*SubdisceptoH, error) {
	subPerms := &models.SubPerms{}
	roleDomain := subRoleDomain(subdiscepto)
	if uH != nil {
		// First, try getting user's permissions
		subPermsMap, err := getUserPerms(ctx, dH.sharedDB, string(roleDomain), uH.id)
		if err != nil {
			return nil, err
		}
		v := models.SubPermsFromMap(subPermsMap)
		subPerms = &v
	}

	// Inherit global perms
	subPerms = &models.SubPerms{
		ReadSubdiscepto:   subPerms.ReadSubdiscepto || dH.globalPerms.ReadSubdiscepto,
		UpdateSubdiscepto: subPerms.UpdateSubdiscepto || dH.globalPerms.UpdateSubdiscepto,
		CreateEssay:       subPerms.CreateEssay || dH.globalPerms.CreateEssay,
		DeleteEssay:       subPerms.DeleteEssay || dH.globalPerms.DeleteEssay,
		BanUser:           subPerms.BanUser || dH.globalPerms.BanUser,
		DeleteSubdiscepto: subPerms.DeleteSubdiscepto || dH.globalPerms.DeleteSubdiscepto,
		ChangeRanking:     subPerms.ChangeRanking || dH.globalPerms.ChangeRanking,
		ManageRole:        subPerms.ManageRole || dH.globalPerms.ManageRole,
		CommonAfterRejoin: subPerms.CommonAfterRejoin || dH.globalPerms.CommonAfterRejoin,
		ViewReport:        subPerms.ViewReport || dH.globalPerms.ViewReport,
		DeleteReport:      subPerms.DeleteReport || dH.globalPerms.DeleteReport,
	}

	if !subPerms.ReadSubdiscepto {
		// Check if the subdiscepto is publicly readable
		public := isSubPublic(ctx, dH.sharedDB, subdiscepto)
		subPerms.ReadSubdiscepto = public
		if !subPerms.ReadSubdiscepto {
			return nil, ErrPermDenied
		}
	}

	rolesH := RolesH{
		contextPerms: subPerms.ToBoolMap(),
		rolesPerms: struct {
			ManageRoles bool
		}{subPerms.ManageRole},
		domain:   roleDomain,
		sharedDB: dH.sharedDB,
	}
	h := &SubdisceptoH{dH.sharedDB, subdiscepto, rolesH, *subPerms}
	return h, nil
}
func (h *SubdisceptoH) Perms() models.SubPerms {
	return h.subPerms
}
func (h *SubdisceptoH) ReadView(ctx context.Context, userH *UserH) (*models.SubdisceptoView, error) {
	if !h.subPerms.ReadSubdiscepto {
		return nil, ErrPermDenied
	}
	return h.readView(ctx, userH)
}
func (h *SubdisceptoH) Delete(ctx context.Context) error {
	fmt.Println(h.subPerms)
	if h.subPerms != models.SubPermsOwner {
		return ErrPermDenied
	}
	return h.deleteSubdiscepto(ctx)
}
func (h *SubdisceptoH) CreateEssay(ctx context.Context, e *models.Essay) (*EssayH, error) {
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
func (h *SubdisceptoH) CreateEssayReply(ctx context.Context, e *models.Essay, pH EssayH) (*EssayH, error) {
	if !h.subPerms.CreateEssay {
		return nil, ErrPermDenied
	}
	e.InReplyTo.Int32 = int32(pH.id)
	e.InReplyTo.Valid = true
	var essay *EssayH
	err := execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		var err error
		essay, err = h.createEssay(ctx, tx, e)
		if err != nil {
			return err
		}
		err = createReply(ctx, tx, e.ID, int(e.InReplyTo.Int32), e.ReplyType.String)
		return err
	})
	if err != nil {
		return nil, err
	}

	// Prepare and send notification
	parentEssayH, err := h.GetEssayH(ctx, int(e.InReplyTo.Int32), nil)
	if err != nil {
		return nil, err
	}
	parentEssay, err := parentEssayH.ReadView(ctx)
	if err != nil {
		return nil, err
	}
	if parentEssay.AttributedToID == e.AttributedToID {
		// Don't notify to self
		return essay, nil
	}
	url, err := url.Parse(fmt.Sprintf("/s/%s/%d", e.PostedIn, e.ID))
	if err != nil {
		return nil, err
	}

	user, err := readPublicUser(ctx, h.sharedDB, e.AttributedToID)
	if err != nil {
		return nil, err
	}
	err = sendNotification(ctx, h.sharedDB, models.Notification{
		UserID:    parentEssay.AttributedToID,
		Title:     user.Name,
		Text:      fmt.Sprintf("replied to your essay"),
		NotifType: models.NotifTypeReply,
		ActionURL: *url,
	})
	if err != nil {
		return nil, err
	}

	return essay, nil
}
func (h *SubdisceptoH) ListAvailablePerms() map[string]bool {
	return models.SubPerms{}.ToBoolMap()
}
func (h *SubdisceptoH) AddMember(ctx context.Context, userH UserH) error {
	if !h.subPerms.ReadSubdiscepto || !userH.perms.Read {
		return ErrPermDenied
	}
	err := addMember(ctx, h.sharedDB, h.name, userH.id)
	if err != nil {
		return rejoin(ctx, h.sharedDB, h.name, userH.id, h.subPerms)
	}
	return nil
}
func (h *SubdisceptoH) RemoveMember(ctx context.Context, userH UserH) error {
	// TODO: should check for specific permission to remove other users
	if !h.subPerms.ReadSubdiscepto || !userH.perms.Read {
		return ErrPermDenied
	}
	return execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		err := removeMember(ctx, h.sharedDB, h.name, userH.id)
		if err != nil {
			return err
		}
		rolesH := h.RolesH.withTx(tx)
		err = rolesH.UnassignAll(ctx, userH.id)
		if err != nil {
			return err
		}
		if h.subPerms.CommonAfterRejoin {
			commonRole, err := rolesH.GetRoleH(ctx, "common-after-rejoin")
			if err == nil {
				return rolesH.Assign(ctx, userH.id, *commonRole)
			}
		}
		return nil
	})
}
func (h *SubdisceptoH) ListReports(ctx context.Context) ([]models.ReportView, error) {
	if !h.subPerms.ViewReport {
		return nil, ErrPermDenied
	}
	sql, args, _ := psql.Select(
		"reports.id",
		"reports.description",
		"essay_view.thesis AS \"essay_view.thesis\"",
		"essay_view.content AS \"essay_view.content\"",
		"essay_view.id AS \"essay_view.id\"",
		"essay_view.posted_in AS \"essay_view.posted_in\"",
		"essay_view.upvotes AS \"essay_view.upvotes\"",
		"essay_view.downvotes AS \"essay_view.downvotes\"",
		"essay_view.attributed_to_name AS \"essay_view.attributed_to_name\"",
	).
		FromSelect(selectEssayWithJoins.
			GroupBy("essays.id", "users.name", "essay_replies.to_id", "essay_replies.reply_type").
			Where(sq.Eq{"essays.posted_in": h.name}),
			"essay_view",
		).
		Join("reports ON essay_view.id = reports.essay_id").
		ToSql()

	reports := []models.ReportView{}
	err := pgxscan.Select(ctx, h.sharedDB, &reports, sql, args...)
	if err != nil {
		return nil, err
	}
	return reports, nil
}
func (h *SubdisceptoH) DeleteReport(ctx context.Context, id int) error {
	if !h.subPerms.DeleteReport {
		return ErrPermDenied
	}
	sql, args, _ := psql.
		Delete("reports").
		Where(sq.Eq{"id": id}).
		ToSql()

	_, err := h.sharedDB.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func (h *SubdisceptoH) Update(ctx context.Context, sub models.Subdiscepto) error {
	if !h.subPerms.UpdateSubdiscepto {
		return ErrPermDenied
	}
	sql, args, _ := psql.
		Update("subdisceptos").
		Set("description", sub.Description).
		Set("questions_required", sub.QuestionsRequired).
		Set("nsfw", sub.Nsfw).
		Set("public", sub.Public).
		Set("min_length", sub.MinLength).
		Where(sq.Eq{"name": h.name}).
		ToSql()

	_, err := h.sharedDB.Exec(ctx, sql, args...)
	return err
}
func createReply(ctx context.Context, db DBTX, fromID int, toID int, replyType string) error {
	_, err := db.Exec(ctx,
		"INSERT INTO essay_replies (from_id, to_id, reply_type) VALUES ($1, $2, $3)",
		fromID, toID, replyType)
	return err
}
func (h *SubdisceptoH) GetEssayH(ctx context.Context, id int, uH *UserH) (*EssayH, error) {
	if !h.subPerms.ReadSubdiscepto {
		return nil, ErrPermDenied
	}
	return h.getEssayH(ctx, id, uH)
}
func (h *SubdisceptoH) getEssayH(ctx context.Context, id int, uH *UserH) (*EssayH, error) {
	// Check if essay is inside this subdiscepto
	sql, args, _ := psql.
		Select("1").
		From("essays").
		Where(sq.Eq{"posted_in": h.name, "id": id}).
		ToSql()

	row := h.sharedDB.QueryRow(ctx, sql, args...)
	var b int
	err := row.Scan(&b)

	if err != nil {
		return nil, err
	}

	isOwner := false
	if uH != nil {
		// Check if user owns the essay
		isOwner = isEssayOwner(ctx, h.sharedDB, id, uH.id)
	}

	// Finally assign capabilities
	essayPerms := models.EssayPerms{
		Read:          h.subPerms.ReadSubdiscepto || isOwner,
		DeleteEssay:   h.subPerms.DeleteEssay || isOwner,
		ChangeRanking: false, // TODO: to implement in future
	}
	e := &EssayH{h.sharedDB, id, essayPerms}
	return e, nil
}
func (h *SubdisceptoH) ListEssays(ctx context.Context) ([]models.EssayView, error) {
	if !h.subPerms.ReadSubdiscepto {
		return nil, ErrPermDenied
	}
	return h.listEssays(ctx)
}
func (h *SubdisceptoH) ListReplies(ctx context.Context, e EssayH, replyType *string) ([]models.EssayView, error) {
	if !h.subPerms.ReadSubdiscepto {
		return nil, ErrPermDenied
	}
	return h.listReplies(ctx, e, replyType)
}
func (h *SubdisceptoH) Name() string {
	return h.name
}

func (h *SubdisceptoH) ListMembers(ctx context.Context) ([]models.Member, error) {
	// Everyone with read access has the right to see who are the moderators
	if !h.subPerms.ReadSubdiscepto {
		return nil, ErrPermDenied
	}

	sqlquery, args, _ := psql.
		Select("subdiscepto_users.user_id", "subdiscepto_users.left_at", "users.name").
		From("subdiscepto_users").
		Join("users ON subdiscepto_users.user_id = users.id").
		Where(sq.Eq{"subdiscepto_users.subdiscepto": h.name}).
		OrderBy("subdiscepto_users.user_id").
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

func (h *SubdisceptoH) createEssay(ctx context.Context, tx DBTX, essay *models.Essay) (*EssayH, error) {
	clen := len(essay.Content)
	subData, err := h.ReadRaw(ctx)
	if err != nil {
		return nil, err
	}
	if clen > LimitMaxContentLen || clen < subData.MinLength {
		return nil, ErrBadContentLen
	}
	essay.PostedIn = h.name
	essay.Published = time.Now()

	err = insertEssay(ctx, tx, essay)
	if err != nil {
		return nil, err
	}
	err = insertTags(ctx, tx, essay.ID, essay.Tags)
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
func insertEssay(ctx context.Context, tx DBTX, essay *models.Essay) error {
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
			essay.PostedIn,
		).
		ToSql()

	row := tx.QueryRow(ctx, sql, args...)
	err := row.Scan(&essay.ID)
	return err
}
func insertTags(ctx context.Context, db DBTX, essayID int, tags []string) error {
	// Insert essay tags
	if len(tags) > LimitMaxTags {
		return ErrTooManyTags
	}

	// Track and skip duplicate tags
	duplicate := make(map[string]bool)

	insertCols := psql.
		Insert("essay_tags").
		Columns("essay_id", "tag")

	for _, tag := range tags {
		if duplicate[tag] {
			continue
		}
		duplicate[tag] = true

		sql, args, _ := insertCols.
			Values(essayID, tag).
			ToSql()

		_, err := db.Exec(ctx,
			sql, args...)
		if err != nil {
			return fmt.Errorf("Error inserting essay_tag in db: %w", err)
		}
	}
	return nil
}
func selectSubdiscepto(userID *int) sq.SelectBuilder {
	return psql.Select(
		"name",
		"description",
		"COUNT(DISTINCT subdiscepto_users.user_id) AS members_count",
	).
		Column("bool_or(CASE subdiscepto_users.user_id WHEN ? THEN true ELSE false END) AS is_member", userID).
		From("subdisceptos").
		LeftJoin("subdiscepto_users ON subdisceptos.name = subdiscepto_users.subdiscepto AND subdiscepto_users.left_at IS NULL").
		GroupBy("subdisceptos.name")
}

func (h *SubdisceptoH) readView(ctx context.Context, userH *UserH) (*models.SubdisceptoView, error) {
	var sub models.SubdisceptoView
	var userID *int
	if userH != nil {
		userID = &userH.id
	}
	sql, args, _ := selectSubdiscepto(userID).
		Where(sq.Eq{"subdisceptos.name": h.name}).
		ToSql()

	err := pgxscan.Get(ctx, h.sharedDB, &sub, sql, args...)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
func (h *SubdisceptoH) ReadRaw(ctx context.Context) (*models.Subdiscepto, error) {
	var sub models.Subdiscepto
	err := pgxscan.Get(ctx, h.sharedDB, &sub, "SELECT * FROM subdisceptos WHERE name = $1", h.name)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
func (h *SubdisceptoH) listEssays(ctx context.Context) ([]models.EssayView, error) {
	var essays []models.EssayView

	sql, args, _ := selectEssayWithJoins.
		GroupBy("essays.id", "users.name", "essay_replies.to_id", "essay_replies.reply_type").
		Where(sq.Eq{"posted_in": h.name}).
		OrderBy("essays.id DESC").
		ToSql()

	err := pgxscan.Select(ctx, h.sharedDB, &essays, sql, args...)
	return essays, err
}
func (h *SubdisceptoH) listReplies(ctx context.Context, e EssayH, replyType *string) (essays []models.EssayView, err error) {
	filterByType := sq.Eq{}
	if replyType != nil {
		filterByType = sq.Eq{"reply_type": replyType}
	}

	sql, args, _ := selectEssay.
		From("essay_replies").
		Join("essays ON essays.id = essay_replies.from_id ").
		LeftJoin("votes ON essays.id = votes.essay_id").
		Join("users ON essays.attributed_to_id = users.id").
		Where(
			sq.And{
				sq.Eq{"essay_replies.to_id": e.id},
				filterByType,
			},
		).
		GroupBy("essays.id", "essay_replies.from_id", "users.name").
		OrderBy("essays.id DESC").
		ToSql()

	err = pgxscan.Select(ctx, h.sharedDB, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
}
func (h *SubdisceptoH) deleteSubdiscepto(ctx context.Context) error {
	return execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		sql, args, _ := psql.
			Delete("subdisceptos").
			Where(sq.Eq{"name": h.name}).
			ToSql()

		_, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}

		sql, args, _ = psql.
			Delete("roles").
			Where("domain = $1", subRoleDomain(h.name)).
			ToSql()

		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}
		return nil
	})
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
	err := execTx(ctx, db, func(ctx context.Context, tx DBTX) error {
		sql, args, _ := psql.
			Insert("subdiscepto_users").
			Columns("subdiscepto", "user_id").
			Values(subdiscepto, userID).
			ToSql()

		_, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}

		commonRole, err := findRoleByName(ctx, tx, string(subRoleDomain(subdiscepto)), "common")
		if err != nil {
			return err
		}
		return assignRole(ctx, tx, userID, commonRole.ID)
	})
	return err
}
func rejoin(ctx context.Context, db DBTX, subdiscepto string, userID int, perms models.SubPerms) error {
	sql, args, _ := psql.
		Update("subdiscepto_users").
		Set("left_at", nil).
		Where(sq.Eq{"user_id": userID, "subdiscepto": subdiscepto}).
		ToSql()

	return execTx(ctx, db, func(ctx context.Context, tx DBTX) error {
		tag, err := tx.Exec(ctx, sql, args...)
		if tag.RowsAffected() == 0 {
			return errors.New("rejoin failed: no rows affected")
		}
		if err != nil {
			return err
		}

		if perms.CommonAfterRejoin {
			roles, err := listUserRoles(ctx, tx, userID, string(subRoleDomain(subdiscepto)))
			if err != nil {
				return err
			}
			rolesMap := map[string]models.Role{}
			for _, r := range roles {
				rolesMap[r.Name] = r
			}
			if _, ok := rolesMap["common-after-rejoin"]; ok {
				err = unassignRole(ctx, tx, userID, rolesMap["common-after-rejoin"].ID)
				if err != nil {
					return err
				}
			}
			if _, ok := rolesMap["common"]; !ok {
				commonRole, err := findRoleByName(ctx, tx, string(subRoleDomain(subdiscepto)), "common")
				if err != nil {
					return err
				}
				return assignRole(ctx, tx, userID, commonRole.ID)
			}
		}
		return nil
	})
}
func removeMember(ctx context.Context, db DBTX, subdiscepto string, userID int) error {
	sql, args, _ := psql.
		Update("subdiscepto_users").
		Set("left_at", "now()").
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
