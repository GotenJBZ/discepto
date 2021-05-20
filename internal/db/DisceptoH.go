package db

import (
	"context"
	"fmt"
	"regexp"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"gitlab.com/ranfdev/discepto/internal/domain"
)

type DisceptoH struct {
	domain.RBACService
	sharedDB    DBTX
	globalPerms domain.GlobalPerms
}

func (sdb *SharedDB) GetDisceptoH(ctx context.Context, uH *UserH, rbacRepo domain.RBACRepo) (*DisceptoH, error) {
	globalPerms := domain.GlobalPerms{}
	
	if uH != nil {
		perms, err := rbacRepo.ListUserPerms(ctx, "discepto", uH.id)
		if err != nil {
			return nil, err
		}
		globalPerms = domain.GlobalPermsFromMap(perms)
		if err != nil {
			return nil, err
		}
	}
	rbacService := domain.NewRBACService(
		rbacRepo,
		globalPerms.ToBoolMap(),
		globalPerms.ManageGlobalRole,
		"discepto",
	)
	return &DisceptoH{globalPerms: globalPerms, RBACService: rbacService, sharedDB: sdb.db}, nil
}

func (h *DisceptoH) Perms() domain.GlobalPerms {
	return h.globalPerms
}
func (h *DisceptoH) ListMembers(ctx context.Context) ([]domain.Member, error) {
	if !h.globalPerms.Login {
		return nil, ErrPermDenied
	}

	sqlquery, args, _ := psql.
		Select("users.id AS user_id", "users.name").
		From("users").
		OrderBy("users.id").
		ToSql()

	members := []domain.Member{}
	err := pgxscan.Select(ctx, h.sharedDB, &members, sqlquery, args...)
	if err != nil {
		return nil, err
	}

	for i := range members {
		members[i].Roles, err = h.ListUserRoles(ctx, members[i].UserID)
	}

	return members, nil
}
func (h *DisceptoH) ReadPublicUser(ctx context.Context, userID int) (*domain.UserView, error) {
	if !h.globalPerms.Login {
		return nil, ErrPermDenied
	}
	return readPublicUser(ctx, h.sharedDB, userID)
}

func (h *DisceptoH) CreateSubdiscepto(ctx context.Context, uH UserH, subd domain.Subdiscepto) (*SubdisceptoH, error) {
	if !h.globalPerms.CreateSubdiscepto {
		return nil, ErrPermDenied
	}
	r := regexp.MustCompile("^\\w+$")
	if !r.Match([]byte(subd.Name)) {
		return nil, ErrInvalidFormat
	}
	return h.createSubdiscepto(ctx, uH, subd)
}
func (h *DisceptoH) ListAvailablePerms() map[string]bool {
	return domain.GlobalPerms{}.ToBoolMap()
}
func (h *DisceptoH) createSubdiscepto(ctx context.Context, uH UserH, subd domain.Subdiscepto) (*SubdisceptoH, error) {
	firstUserID := uH.id
	subH := &SubdisceptoH{
		sharedDB: h.sharedDB,
		name:     subd.Name,
		RolesH: RolesH{
			contextPerms: domain.SubPermsOwner.ToBoolMap(),
			rolesPerms:   struct{ ManageRoles bool }{true},
			domain:       subRoleDomain(subd.Name),
			sharedDB:     h.sharedDB,
		},
		subPerms: domain.SubPermsOwner,
	}
	err := execTx(ctx, h.sharedDB, func(ctx context.Context, tx DBTX) error {
		err := createSubdiscepto(ctx, tx, subd)
		if err != nil {
			return err
		}

		// Create a "common" role, added to every user of the subdiscepto
		subPerms := domain.SubPerms{
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
		_, err = createRole(ctx, tx, string(subH.RolesH.domain), "common", false, p)
		if err != nil {
			return err
		}

		// Create a "common-after-rejoin" role, added to every user while away from the subdiscepto
		subPerms = domain.SubPerms{
			CommonAfterRejoin: true,
		}
		p = subPerms.ToBoolMap()
		_, err = createRole(ctx, tx, string(subH.RolesH.domain), "common-after-rejoin", false, p)
		if err != nil {
			return err
		}

		// Create an "admin" role, added to the first user
		subPerms = domain.SubPerms{
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
		adminRoleID, err := createRole(ctx, tx, string(subH.RolesH.domain), "admin", true, subPerms.ToBoolMap())
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
	return subH, nil
}
func (h *DisceptoH) DeleteReport(ctx context.Context, report *domain.Report) error {
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
func (h *DisceptoH) ListRecentEssaysIn(ctx context.Context, subs []string) ([]domain.EssayView, error) {
	essayPreviews := []domain.EssayView{}
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
func (sdb *SharedDB) ListSubdisceptos(ctx context.Context, userH *UserH) ([]domain.SubdisceptoView, error) {
	var subs []domain.SubdisceptoView
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
func (h *DisceptoH) SearchByTags(ctx context.Context, tags []string) ([]domain.EssayView, error) {
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
func scanEssays(ctx context.Context, rows pgx.Rows, tags []string) ([]domain.EssayView, error) {
	essayMap := map[int]*domain.EssayView{}
	for rows.Next() {
		tmp := &domain.EssayRow{}
		err := pgxscan.ScanRow(tmp, rows)
		if err != nil {
			return nil, err
		}
		if essay, ok := essayMap[tmp.ID]; ok {
			essay.Tags = append(essay.Tags, tmp.Tag)
		} else {
			essayMap[tmp.ID] = &domain.EssayView{
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
				Replying: domain.Replying{
					InReplyTo: tmp.InReplyTo,
					ReplyType: tmp.ReplyType,
				},
			}
		}
	}
	n := rows.CommandTag().RowsAffected()
	essays := make([]domain.EssayView, 0, n)
	for _, v := range essayMap {
		essays = append(essays, *v)
	}

	return essays, nil
}
func (h *DisceptoH) SearchByThesis(ctx context.Context, title string) ([]domain.EssayView, error) {
	sql, args, _ := selectEssayWithJoins.
		Join("subdisceptos ON subdisceptos.name = essays.posted_in").
		Where("subdisceptos.public = true AND essays.thesis ILIKE $1", fmt.Sprintf(`%%%s%%`, title)).
		GroupBy("essays.id", "essay_replies.from_id", "users.name").
		OrderBy("essays.id DESC").
		ToSql()

	essays := []domain.EssayView{}

	err := pgxscan.Select(ctx, h.sharedDB, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
}
func (h *DisceptoH) ListUserEssays(ctx context.Context, userID int) ([]domain.EssayView, error) {
	return listUserEssays(ctx, h.sharedDB, userID)
}
func readPublicUser(ctx context.Context, db DBTX, userID int) (*domain.UserView, error) {
	user := &domain.UserView{}
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
