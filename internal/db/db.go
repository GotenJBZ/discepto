package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

const (
	LimitMaxTags       = 10
	LimitMinContentLen = 150
	LimitMaxContentLen = 5000 // 5K
	TokenLen           = 64   // 64 bytes
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var ErrBadEmailSyntax error = errors.New("Bad email syntax")
var ErrTooManyTags error = errors.New("You have inserted too many tags")
var ErrBadContentLen error = errors.New("You have to respect the imposed content length limits")
var ErrEmailAlreadyUsed error = errors.New("The email is already used")

type DB struct {
	db         *pgxpool.Pool
	config     *models.EnvConfig
	bcryptCost int
}

func Connect(config *models.EnvConfig) (DB, error) {
	db, err := pgxpool.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		err = fmt.Errorf("Failed to connect to postgres: %w", err)
	}
	bcryptCost := bcrypt.DefaultCost + 2
	if config.Debug {
		bcryptCost = bcrypt.MinCost
	}
	return DB{
		db,
		config,
		bcryptCost,
	}, err
}

func MigrateUp(dbURL string) error {
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		return fmt.Errorf("Error reading migrations: %s", err)
	}
	defer m.Close()
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("While migrating up: %s", err)
	}
	return nil
}
func MigrateDown(dbURL string) error {
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		return fmt.Errorf("Error reading migrations: %s", err)
	}
	defer m.Close()
	err = m.Down()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("While migrating down: %s", err)
	}
	return nil
}
func Drop(dbURL string) error {
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		return fmt.Errorf("Error reading migrations: %s", err)
	}
	defer m.Close()
	err = m.Drop()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("While dropping: %s", err)
	}
	return nil
}

func (db *DB) ListUsers() ([]models.User, error) {
	var users []models.User
	err := pgxscan.Select(context.Background(), db.db, &users, "SELECT id, name, email FROM users")
	return users, err
}

func (db *DB) CreateUser(user *models.User, passwd string) (err error) {
	// Check email format
	if !utils.ValidateEmail(user.Email) {
		return ErrBadEmailSyntax
	}

	// Check if email is already used
	var exists bool
	err = pgxscan.Get(context.Background(),
		db.db,
		&exists,
		"SELECT exists(SELECT 1 FROM users WHERE email = $1)",
		user.Email)

	if err != nil {
		return err
	}
	if exists {
		return ErrEmailAlreadyUsed
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(passwd), db.bcryptCost)

	// Insert the new user
	sql, args, _ := psql.
		Insert("users").
		Columns("name", "email", "passwd_hash").
		Values(user.Name, user.Email, hash).
		Suffix("RETURNING id").
		ToSql()

	row := db.db.QueryRow(context.Background(), sql, args...)
	err = row.Scan(&user.ID)
	if err != nil {
		return err
	}

	return nil
}
func (db *DB) Login(email string, passwd string) (token string, err error) {
	sql, args, _ := psql.
		Select("id", "passwd_hash").
		From("users").
		Where(sq.Eq{"email": email}).
		ToSql()

	var data struct {
		ID         int
		PasswdHash string
	}
	err = pgxscan.Get(
		context.Background(),
		db.db,
		&data,
		sql,
		args...,
	)
	if err != nil {
		return "", err
	}
	compareErr := bcrypt.CompareHashAndPassword([]byte(data.PasswdHash), []byte(passwd))
	if compareErr != nil {
		return "", compareErr
	}

	// Insert a new token
	token = utils.GenToken(TokenLen)
	sql, args, _ = psql.
		Insert("tokens").
		Columns("user_id", "token").
		Values(data.ID, token).
		ToSql()

	_, err = db.db.Exec(context.Background(), sql, args...)
	if err != nil {
		return "", err
	}
	return token, nil
}
func (db *DB) Signout(token string) error {
	_, err := db.db.Exec(context.Background(), "DELETE FROM tokens WHERE tokens.token = $1", token)
	if err != nil {
		return err
	}
	return nil
}
func (db *DB) GetUserByToken(token string) (*models.User, error) {
	user := &models.User{}
	sql, args, _ := psql.
		Select("users.name", "users.id", "users.email").
		From("users").
		LeftJoin("tokens ON users.id = tokens.user_id").
		Where(sq.Eq{"tokens.token": token}).
		ToSql()

	err := pgxscan.Get(
		context.Background(),
		db.db, user,
		sql, args...)

	if err != nil {
		return nil, err
	}
	return user, nil
}

func (db *DB) DeleteUser(id int) error {
	sql, args, _ := psql.Delete("users").Where(sq.Eq{"id": id}).ToSql()
	_, err := db.db.Exec(context.Background(), sql, args...)
	return err
}
func (db *DB) ListEssays(subName string) ([]*models.Essay, error) {
	var essays []*models.Essay

	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{"posted_in": subName}).
		ToSql()

	err := pgxscan.Select(context.Background(), db.db, &essays, sql, args...)
	return essays, err
}
func (db *DB) CreateEssay(essay *models.Essay) error {
	clen := len(essay.Content)
	if clen > LimitMaxContentLen || clen < LimitMinContentLen {
		return ErrBadContentLen
	}

	tx, err := db.db.Begin(context.Background())
	defer tx.Rollback(context.Background())
	if err != nil {
		return err
	}

	// Insert essay
	sql, args, _ := psql.
		Insert("essays").
		Columns("thesis", "content", "attributed_to_id", "published", "posted_in", "in_reply_to", "reply_type").
		Suffix("RETURNING id").
		Values(
			essay.Thesis,
			essay.Content,
			essay.AttributedToID,
			essay.Published,
			essay.PostedIn,
			essay.InReplyTo,
			essay.ReplyType,
		).
		ToSql()

	row := tx.QueryRow(context.Background(), sql, args...)
	err = row.Scan(&essay.ID)
	if err != nil {
		return fmt.Errorf("Error inserting essay in db: %w", err)
	}

	err = db.insertTags(tx, essay)
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}
	return nil
}
func (db *DB) insertTags(tx pgx.Tx, essay *models.Essay) error {
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

		_, err := tx.Exec(context.Background(),
			sql, args...)
		if err != nil {
			return fmt.Errorf("Error inserting essay_tag in db: %w", err)
		}
	}
	return nil
}
func (db *DB) GetEssay(id int) (*models.Essay, error) {
	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{"id": id}).
		ToSql()

	var essay models.Essay
	err := pgxscan.Get(context.Background(), db.db, &essay, sql, args...)
	if err != nil {
		return nil, err
	}

	sql, args, _ = psql.
		Select("tag").
		From("essay_tags").
		Where(sq.Eq{"essay_id": id}).
		ToSql()
	err = pgxscan.Select(context.Background(), db.db, &essay.Tags, sql, args...)
	if err != nil {
		return nil, err
	}

	return &essay, nil
}
func (db *DB) DeleteEssay(id int) error {
	sql, args, _ := psql.Delete("essays").Where(sq.Eq{"id": id}).ToSql()
	_, err := db.db.Exec(context.Background(), sql, args...)
	return err
}
func (db *DB) CreateSubdiscepto(subd *models.Subdiscepto, firstUserID int) error {
	tx, err := db.db.Begin(context.Background())
	if err != nil {
		return err
	}

	// Insert subdiscepto
	sql, args, _ := psql.
		Insert("subdisceptos").
		Columns("name", "description", "min_length", "questions_required", "nsfw", "owner_id").
		Values(subd.Name, subd.Description, subd.MinLength, subd.QuestionsRequired, subd.Nsfw, firstUserID).
		ToSql()

	_, err = tx.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}

	// Insert first user of subdiscepto
	sql, args, _ = psql.
		Insert("subdiscepto_users").
		Columns("subdiscepto", "user_id").
		Values(subd.Name, firstUserID).
		ToSql()

	_, err = tx.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}

	// Insert first user of subdiscepto
	sql, args, _ = psql.
		Insert("user_preset_sub_roles").
		Columns("subdiscepto", "user_id", "role_name").
		Values(subd.Name, firstUserID, "admin").
		ToSql()

	_, err = tx.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}
	return nil
}
func (db *DB) GetSubdiscepto(name string) (*models.Subdiscepto, error) {
	var sub models.Subdiscepto
	err := pgxscan.Get(context.Background(), db.db, &sub, "SELECT * FROM subdisceptos WHERE name = $1", name)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
func (db *DB) ListSubdisceptos() ([]*models.Subdiscepto, error) {
	var subs []*models.Subdiscepto
	err := pgxscan.Select(context.Background(), db.db, &subs, "SELECT name, description, min_length, questions_required, nsfw FROM subdisceptos")
	if err != nil {
		return nil, err
	}
	return subs, nil
}
func (db *DB) JoinSubdiscepto(sub string, userID int) error {
	sql, args, _ := psql.
		Insert("subdiscepto_users").
		Columns("subdiscepto", "user_id").
		Values(sub, userID).
		ToSql()

	_, err := db.db.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func (db *DB) LeaveSubdiscepto(sub string, userID int) error {
	sql, args, _ := psql.
		Delete("subdiscepto_users").
		Where(sq.Eq{"subdiscepto": sub, "user_id": userID}).
		ToSql()

	_, err := db.db.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func (db *DB) ListMySubdisceptos(userID int) (subs []string, err error) {
	sql, args, _ := psql.
		Select("subdiscepto").
		From("subdiscepto_users").
		Where(sq.Eq{"user_id": userID}).
		ToSql()

	err = pgxscan.Select(context.Background(), db.db, &subs, sql, args...)
	if err != nil {
		return nil, err
	}
	return subs, nil
}
func (db *DB) ListRecentEssaysIn(subs []string) (essays []*models.Essay, err error) {
	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{"posted_in": subs}).
		ToSql()

	err = pgxscan.Select(context.Background(), db.db, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
}
func (db *DB) ListEssayReplies(essayID int, replyType string) (essays []*models.Essay, err error) {
	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{
			"in_reply_to": essayID,
			"reply_type":  replyType,
		}).
		ToSql()

	err = pgxscan.Select(context.Background(), db.db, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
}
func (db *DB) DeleteSubdiscepto(name string) error {
	sql, args, _ := psql.
		Delete("subdisceptos").
		Where(sq.Eq{"name": name}).
		ToSql()

	_, err := db.db.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func (db *DB) CreateReport(report *models.Report) error {
	sql, args, _ := psql.
		Insert("reports").
		Columns("flag", "essay_id", "from_user_id", "to_user_id").
		Values(report.Flag, report.EssayID, report.FromUserID, report.ToUserID).
		Suffix("RETURNING id").
		ToSql()
	row := db.db.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&report.ID)
	if err != nil {
		return err
	}
	return nil
}

//TODO: Add on delete cascade to schema
func (db *DB) DeleteReport(report *models.Report) error {
	sql, args, _ := psql.
		Delete("reports").
		Where(sq.Eq{"id": report.ID}).
		ToSql()

	_, err := db.db.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func (db *DB) CreateVote(vote *models.Vote) error {
	sql, args, _ := psql.
		Insert("votes").
		Columns("user_id", "essay_id", "vote_type").
		Values(vote.UserID, vote.EssayID, vote.VoteType).
		ToSql()

	_, err := db.db.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func (db *DB) CountVotes(essayID int) (upvotes, downvotes int, err error) {
	sql, args, _ := psql.
		Select("COUNT(*), SUM(CASE WHEN vote_type = 'upvote' THEN 1 ELSE 0 END) AS upvotes").
		From("votes").
		Where(sq.Eq{"essay_id": essayID}).
		GroupBy("vote_type").
		ToSql()

	row := db.db.QueryRow(context.Background(), sql, args...)
	var total int
	err = row.Scan(&total, &upvotes)
	if err == pgx.ErrNoRows {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}

	return upvotes, total - upvotes, nil
}
func (db *DB) DeleteVote(essayID, userID int) error {
	sql, args, _ := psql.
		Delete("votes").
		Where(sq.Eq{"user_id": userID, "essay_id": essayID}).
		ToSql()
	_, err := db.db.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func (db *DB) SearchByTags(tags []string) (essays []*models.Essay, err error) {
	sql, args, _ := psql.
		Select("thesis", "content", "reply_type").
		Distinct().
		From("essays").
		LeftJoin("essay_tags ON id = essay_id").
		Where(sq.Eq{"tag": tags}).
		ToSql()

	err = pgxscan.Select(context.Background(), db.db, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
}
func (db *DB) GetSubPerms(userID int, subName string) (perms models.SubPerms, err error) {
	queryPresetSubRoles := sq.Select("sub_perms_id").
		From("user_preset_sub_roles").
		Join("preset_sub_roles ON user_preset_sub_roles.role_name = preset_sub_roles.name").
		Where(sq.Eq{"user_preset_sub_roles.subdiscepto": subName, "user_id": userID})

	queryCustomSubPerms := sq.Select("sub_perms_id").
		From("user_custom_sub_roles").
		Join("custom_sub_roles ON user_custom_sub_roles.role_name = custom_sub_roles.name AND user_custom_sub_roles.subdiscepto = custom_sub_roles.subdiscepto").
		Where(sq.Eq{"custom_sub_roles.subdiscepto": subName, "user_id": userID})

	queryAllSubPerms := queryPresetSubRoles.Suffix("UNION").SuffixExpr(queryCustomSubPerms)

	bool_or := func(col string) string {
		return fmt.Sprintf("bool_or(%s) AS %s", col, col)
	}

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

	fmt.Println(sql, args)

	err = pgxscan.Get(context.Background(), db.db, &perms, sql, args...)
	return perms, err
}
