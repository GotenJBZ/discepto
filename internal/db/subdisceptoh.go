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

type SubdisceptoH struct {
	RolesH
	sharedDB     DBTX
	rawSub       *models.Subdiscepto
	subPerms     models.Perms
	notifService models.NotificationService
}

func (h *SubdisceptoH) Perms() models.Perms {
	return h.subPerms
}
func (h *SubdisceptoH) ReadView(ctx context.Context, userH *UserH) (*models.SubdisceptoView, error) {
	if err := h.subPerms.Require(models.PermReadSubdiscepto); err != nil {
		return nil, err
	}
	return h.readView(ctx, userH)
}
func (h *SubdisceptoH) Delete(ctx context.Context) error {
	fmt.Println(h.subPerms)
	if err := h.subPerms.Require(models.PermDeleteSubdiscepto); err != nil {
		return err
	}
	return h.deleteSubdiscepto(ctx)
}
func (h *SubdisceptoH) CreateEssay(ctx context.Context, e *models.Essay) (*EssayH, error) {
	if err := h.subPerms.Require(models.PermCreateEssay); err != nil {
		return nil, err
	}
	if e.InReplyTo.Valid {
		return nil, fmt.Errorf("can't reply with method CreateEssay")
	}
	var essay *EssayH
	return essay, execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		var err error
		essay, err = h.createEssay(ctx, tx, e)
		return err
	})
}
func (h *SubdisceptoH) CreateEssayReply(ctx context.Context, e *models.Essay, pH EssayH) (*EssayH, error) {
	if err := h.subPerms.Require(models.PermCreateEssay); err != nil {
		return nil, err
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
	err = h.notifService.Send(ctx, &models.Notification{
		Title:     user.Name,
		Text:      "replied to your essay",
		NotifType: models.NotifTypeReply,
		ActionURL: *url,
	}, parentEssay.AttributedToID)
	if err != nil {
		return nil, err
	}

	return essay, nil
}
func (h *SubdisceptoH) ListAvailablePerms() models.Perms {
	return models.PermsSubAdmin
}
func (h *SubdisceptoH) AddMember(ctx context.Context, userH UserH) error {
	if !userH.perms.Read {
		return models.ErrPermDenied
	}
	if err := h.subPerms.Require(models.PermReadSubdiscepto); err != nil {
		return err
	}
	err := addMember(ctx, h.sharedDB, h.rawSub, userH.id)
	if err != nil {
		return rejoin(ctx, h.sharedDB, h.rawSub, userH.id, h.subPerms)
	}
	return nil
}
func (h *SubdisceptoH) RemoveMember(ctx context.Context, userH UserH) error {
	if !userH.perms.Read {
		return models.ErrPermDenied
	}
	if err := h.subPerms.Require(models.PermReadSubdiscepto); err != nil {
		return err
	}
	return execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		err := removeMember(ctx, h.sharedDB, h.rawSub, userH.id)
		if err != nil {
			return err
		}
		rolesH := newUnsafeRolesH(h.sharedDB, h.subPerms, h.domain).withTx(tx)
		err = rolesH.UnassignAll(ctx, userH.id)
		if err != nil {
			return err
		}
		if err := h.subPerms.Require(models.PermCommonAfterRejoin); err == nil {
			commonAfterRejoin, err := rolesH.GetRoleH(ctx, "common-after-rejoin")
			if err != nil {
				return err
			}
			return rolesH.Assign(ctx, userH.id, *commonAfterRejoin)
		}
		return nil
	})
}

func (h *SubdisceptoH) ListReports(ctx context.Context) ([]models.ReportView, error) {
	if err := h.subPerms.Require(models.PermViewReport); err != nil {
		return nil, err
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
			Where(sq.Eq{"essays.posted_in": h.rawSub.Name}),
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
	if err := h.subPerms.Require(models.PermDeleteReport); err != nil {
		return err
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
func (h *SubdisceptoH) Update(ctx context.Context, subReq *models.SubdisceptoReq) error {
	if err := h.subPerms.Require(models.PermUpdateSubdiscepto); err != nil {
		return err
	}
	sql, args, _ := psql.
		Update("subdisceptos").
		Set("description", subReq.Description).
		Set("questions_required", subReq.QuestionsRequired).
		Set("nsfw", subReq.Nsfw).
		Set("public", subReq.Public).
		Set("min_length", subReq.MinLength).
		Where(sq.Eq{"name": h.rawSub.Name}).
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
	if err := h.subPerms.Require(models.PermReadSubdiscepto); err != nil {
		return nil, err
	}
	return h.getEssayH(ctx, id, uH)
}
func (h *SubdisceptoH) getEssayH(ctx context.Context, id int, uH *UserH) (*EssayH, error) {
	// Check if essay is inside this subdiscepto
	sql, args, _ := psql.
		Select("1").
		From("essays").
		Where(sq.Eq{"posted_in": h.rawSub.Name, "id": id}).
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

	essayPerms := h.subPerms

	if isOwner {
		essayPerms = essayPerms.Union(models.NewPerms(models.PermDeleteEssay))
	}

	// Finally assign capabilities
	e := &EssayH{h.sharedDB, id, essayPerms, h.notifService}
	return e, nil
}
func (h *SubdisceptoH) ListEssays(ctx context.Context) ([]models.EssayView, error) {
	if err := h.subPerms.Require(models.PermReadSubdiscepto); err != nil {
		return nil, err
	}
	return h.listEssays(ctx)
}
func (h *SubdisceptoH) ListReplies(ctx context.Context, e EssayH, replyType *string) ([]models.EssayView, error) {
	if err := h.subPerms.Require(models.PermReadSubdiscepto); err != nil {
		return nil, err
	}
	return h.listReplies(ctx, e, replyType)
}
func (h *SubdisceptoH) Name() string {
	return h.rawSub.Name
}
func (h *SubdisceptoH) RoleDomain() models.RoleDomain {
	return h.rawSub.RoledomainID
}

func (h *SubdisceptoH) ListMembers(ctx context.Context) ([]models.Member, error) {
	// Everyone with read access has the right to see who are the moderators
	if err := h.subPerms.Require(models.PermReadSubdiscepto); err != nil {
		return nil, err
	}

	sqlquery, args, _ := psql.
		Select("subdiscepto_users.user_id", "subdiscepto_users.left_at", "users.name").
		From("subdiscepto_users").
		Join("users ON subdiscepto_users.user_id = users.id").
		Where(sq.Eq{"subdiscepto_users.subdiscepto": h.rawSub.Name}).
		OrderBy("subdiscepto_users.user_id").
		ToSql()

	members := []models.Member{}
	err := pgxscan.Select(ctx, h.sharedDB, &members, sqlquery, args...)
	if err != nil {
		return nil, err
	}

	for i := range members {
		members[i].Roles, _ = h.ListUserRoles(ctx, members[i].UserID)
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
		return nil, models.ErrBadContentLen
	}
	essay.PostedIn = h.rawSub.Name
	essay.Published = time.Now()

	err = insertEssay(ctx, tx, essay)
	if err != nil {
		return nil, err
	}
	err = insertTags(ctx, tx, essay.ID, essay.Tags)
	if err != nil {
		return nil, err
	}
	essayPerms := h.subPerms.Union(models.NewPerms(models.PermDeleteEssay))

	return &EssayH{h.sharedDB, essay.ID, essayPerms, h.notifService}, err
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
		return models.ErrTooManyTags
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
			return fmt.Errorf("error inserting essay_tag in db: %w", err)
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
		Where(sq.Eq{"subdisceptos.name": h.rawSub.Name}).
		ToSql()

	err := pgxscan.Get(ctx, h.sharedDB, &sub, sql, args...)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
func (h *SubdisceptoH) ReadRaw(ctx context.Context) (*models.Subdiscepto, error) {
	if err := h.subPerms.Require(models.PermReadSubdiscepto); err != nil {
		return nil, err
	}
	return readRawSub(ctx, h.sharedDB, h.rawSub.Name)
}
func (h *SubdisceptoH) listEssays(ctx context.Context) ([]models.EssayView, error) {
	var essays []models.EssayView

	sql, args, _ := selectEssayWithJoins.
		GroupBy("essays.id", "users.name", "essay_replies.to_id", "essay_replies.reply_type").
		Where(sq.Eq{"posted_in": h.rawSub.Name}).
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
			Where(sq.Eq{"name": h.rawSub.Name}).
			ToSql()

		_, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}

		sql, args, _ = psql.
			Delete("roledomains").
			Where("id = $1", h.rawSub.RoledomainID).
			ToSql()

		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}
		return nil
	})
}

func insertMember(ctx context.Context, db DBTX, rawSub *models.Subdiscepto, userID int) error {
	sql, args, _ := psql.
		Insert("subdiscepto_users").
		Columns("subdiscepto", "user_id").
		Values(rawSub.Name, userID).
		ToSql()

	_, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}

func addMember(ctx context.Context, db DBTX, rawSub *models.Subdiscepto, userID int) error {
	err := execTx(ctx, db, func(ctx context.Context, tx DBTX) error {
		err := insertMember(ctx, db, rawSub, userID)
		if err != nil {
			return err
		}

		commonRole, err := findRoleByName(ctx, tx, rawSub.RoledomainID, "common")
		if err != nil {
			return err
		}
		return assignRole(ctx, tx, userID, commonRole.ID)
	})
	return err
}
func rejoin(ctx context.Context, db DBTX, rawSub *models.Subdiscepto, userID int, perms models.Perms) error {
	sql, args, _ := psql.
		Update("subdiscepto_users").
		Set("left_at", nil).
		Where(sq.Eq{"user_id": userID, "subdiscepto": rawSub.Name}).
		ToSql()

	return execTx(ctx, db, func(ctx context.Context, tx DBTX) error {
		tag, err := tx.Exec(ctx, sql, args...)
		if tag.RowsAffected() == 0 {
			return errors.New("rejoin failed: no rows affected")
		}
		if err != nil {
			return err
		}

		if err := perms.Require(models.PermCommonAfterRejoin); err == nil {
			roles, err := listUserRoles(ctx, tx, userID, rawSub.RoledomainID)
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
				commonRole, err := findRoleByName(ctx, tx, rawSub.RoledomainID, "common")
				if err != nil {
					return err
				}
				return assignRole(ctx, tx, userID, commonRole.ID)
			}
		}
		return nil
	})
}
func removeMember(ctx context.Context, db DBTX, sub *models.Subdiscepto, userID int) error {
	sql, args, _ := psql.
		Update("subdiscepto_users").
		Set("left_at", "now()").
		Where(sq.Eq{"subdiscepto": sub.Name, "user_id": userID}).
		ToSql()

	_, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func insertSubdiscepto(ctx context.Context, db DBTX, sub models.Subdiscepto) error {
	// Insert subdiscepto
	sql, args, _ := psql.
		Insert("subdisceptos").
		Columns("name",
			"description",
			"min_length",
			"questions_required",
			"nsfw",
			"public",
			"roledomain_id").
		Values(sub.Name,
			sub.Description,
			sub.MinLength,
			sub.QuestionsRequired,
			sub.Nsfw,
			sub.Public,
			sub.RoledomainID).
		ToSql()
	_, err := db.Exec(ctx, sql, args...)
	return err
}
func readRawSub(ctx context.Context, db DBTX, name string) (*models.Subdiscepto, error) {
	var sub models.Subdiscepto
	err := pgxscan.Get(ctx, db, &sub, "SELECT * FROM subdisceptos WHERE name = $1", name)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
