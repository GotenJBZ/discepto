package db

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/Masterminds/squirrel"
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

var BCryptCost = 11

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

var ErrBadEmailSyntax error = errors.New("Bad email syntax")
var ErrTooManyTags error = errors.New("You have inserted too many tags")
var ErrBadContentLen error = errors.New("You have to respect the imposed content length limits")
var ErrEmailAlreadyUsed error = errors.New("The email is already used")

func init() {
	if os.Getenv("DEBUG") == "true" {
		BCryptCost = bcrypt.MinCost // To speed up testing
	}
}

var DB *pgxpool.Pool

func CheckEnvDatabaseUrl() string {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		panic(errors.New("DATABASE_URL env variable missing"))
	}
	return dbUrl
}
func Connect() error {
	dbUrl := CheckEnvDatabaseUrl()
	// dbUrl := "postgres://discepto:passwd@localhost/disceptoDb"
	var err error = nil
	DB, err = pgxpool.Connect(context.Background(), dbUrl)
	if err != nil {
		err = fmt.Errorf("Failed to connect to postgres: %w", err)
	}
	return err
}

func MigrateUp() error {
	dbUrl := CheckEnvDatabaseUrl()
	m, err := migrate.New("file://migrations", dbUrl)
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
func MigrateDown() error {
	dbUrl := CheckEnvDatabaseUrl()
	m, err := migrate.New("file://migrations", dbUrl)
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
func Drop() error {
	dbUrl := CheckEnvDatabaseUrl()
	m, err := migrate.New("file://migrations", dbUrl)
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

func ListUsers() ([]models.User, error) {
	var users []models.User
	err := pgxscan.Select(context.Background(), DB, &users, "SELECT * FROM users")
	return users, err
}

func CreateUser(user *models.User, passwd string) (err error) {
	// Check email format
	if !utils.ValidateEmail(user.Email) {
		return ErrBadEmailSyntax
	}

	// Check if email is already used
	var exists bool
	err = pgxscan.Get(context.Background(),
		DB,
		&exists,
		"SELECT exists(SELECT 1 FROM users WHERE email = $1)",
		user.Email)

	if err != nil {
		return err
	}
	if exists {
		return ErrEmailAlreadyUsed
	}

	// Insert the new user
	tx, err := DB.Begin(context.Background())
	sql, args, _ := psql.
		Insert("users").
		Columns("name", "email", "role_id").
		Values(user.Name, user.Email, user.RoleID).
		Suffix("RETURNING id").
		ToSql()
	row := tx.QueryRow(context.Background(), sql, args...)
	err = row.Scan(&user.ID)
	if err != nil {
		return err
	}

	// Insert the password hash
	hash, err := bcrypt.GenerateFromPassword([]byte(passwd), BCryptCost)
	sql, args, _ = psql.
		Insert("credentials").
		Columns("user_id", "hash").
		Values(user.ID, string(hash)).
		ToSql()

	_, err = tx.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}

	// Commit changes
	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}
	return nil
}
func Login(email string, passwd string) (token string, err error) {
	type res struct {
		Hash string
		ID   int
	}
	sql, args, _ := psql.
		Select("credentials.hash, users.id").
		From("credentials").
		LeftJoin("users ON users.id = credentials.user_id").
		Where("users.email = $1", email).
		ToSql()

	var data res
	err = pgxscan.Get(
		context.Background(),
		DB,
		&data,
		sql,
		args...,
	)
	if err != nil {
		return "", err
	}
	compareErr := bcrypt.CompareHashAndPassword([]byte(data.Hash), []byte(passwd))
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

	_, err = DB.Exec(context.Background(), sql, args...)
	if err != nil {
		return "", err
	}
	return token, nil
}
func Signout(token string) error {
	_, err := DB.Exec(context.Background(), "DELETE FROM tokens WHERE tokens.token = $1", token)
	if err != nil {
		return err
	}
	return nil
}
func GetUserByToken(token string) (*models.User, error) {
	user := &models.User{}
	sql, args, _ := psql.
		Select("users.name", "users.id", "users.role_id", "users.email").
		From("users").
		LeftJoin("tokens ON users.id = tokens.user_id").
		Where("tokens.token = $1", token).
		ToSql()

	err := pgxscan.Get(
		context.Background(),
		DB, user,
		sql, args...)

	if err != nil {
		return nil, err
	}
	return user, nil
}

var roleQuery = psql.
	Select("roles.name", "roles.permissions").
	From("roles").
	LeftJoin("users ON roles.id = users.role_id")

func GetGlobalRole(userID int) (role *models.Role, err error) {
	sql, args, _ := roleQuery.
		Where("users.id = $1", userID).ToSql()

	role = &models.Role{}
	err = pgxscan.Get(context.Background(), DB, role, sql, args...)
	if err != nil {
		return nil, err
	}
	return role, nil
}
func DeleteUser(id int) error {
	sql, args, _ := psql.Delete("users").Where("id = $1", id).ToSql()
	_, err := DB.Exec(context.Background(), sql, args...)
	return err
}
func ListEssays(subName string) ([]*models.Essay, error) {
	var essays []*models.Essay

	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where("posted_in = $1", subName).
		ToSql()

	err := pgxscan.Select(context.Background(), DB, &essays, sql, args...)
	return essays, err
}
func CreateEssay(essay *models.Essay) error {
	clen := len(essay.Content)
	if clen > LimitMaxContentLen || clen < LimitMinContentLen {
		return ErrBadContentLen
	}

	tx, err := DB.Begin(context.Background())
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

	err = insertTags(tx, essay)
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}
	return nil
}
func insertTags(tx pgx.Tx, essay *models.Essay) error {
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
func GetEssay(id int) (*models.Essay, error) {
	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where("id = $1", id).
		ToSql()

	var essay models.Essay
	err := pgxscan.Get(context.Background(), DB, &essay, sql, args...)
	if err != nil {
		return nil, err
	}

	sql, args, _ = psql.
		Select("tag").
		From("essay_tags").
		Where("essay_id = $1", id).
		ToSql()
	err = pgxscan.Select(context.Background(), DB, &essay.Tags, sql, args...)
	if err != nil {
		return nil, err
	}

	return &essay, nil
}
func DeleteEssay(id int) error {
	sql, args, _ := psql.Delete("essays").Where("id = $1", id).ToSql()
	_, err := DB.Exec(context.Background(), sql, args...)
	return err
}
func CreateSubdiscepto(subd *models.Subdiscepto, firstUserID int) error {
	tx, err := DB.Begin(context.Background())
	if err != nil {
		return err
	}

	// Insert subdiscepto
	sql, args, _ := psql.
		Insert("subdisceptos").
		Columns("name", "description").
		Values(subd.Name, subd.Description).
		ToSql()

	_, err = tx.Exec(context.Background(), sql, args...)

	// Insert first user of subdiscepto
	sql, args, _ = psql.
		Insert("subdiscepto_users").
		Columns("name", "user_id", "role_id").
		Values(subd.Name, firstUserID, models.RoleAdmin).
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
func GetSubdiscepto(name string) (*models.Subdiscepto, error) {
	var sub models.Subdiscepto
	err := pgxscan.Get(context.Background(), DB, &sub, "SELECT * FROM subdisceptos WHERE name = $1", name)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
func ListSubdisceptos() ([]*models.Subdiscepto, error) {
	var subs []*models.Subdiscepto
	err := pgxscan.Select(context.Background(), DB, &subs, "SELECT * FROM subdisceptos")
	if err != nil {
		return nil, err
	}
	return subs, nil
}
func JoinSubdiscepto(sub string, userID int) error {
	sql, args, _ := psql.
		Insert("subdiscepto_users").
		Columns("name", "user_id").
		Values(sub, userID).
		ToSql()

	_, err := DB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func LeaveSubdiscepto(sub string, userID int) error {
	sql, args, _ := psql.
		Delete("subdiscepto_users").
		Where("name = $1 AND user_id = $2", sub, userID).
		ToSql()

	_, err := DB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func ListMySubdisceptos(userID int) (subs []string, err error) {
	sql, args, _ := psql.
		Select("name").
		From("subdiscepto_users").
		Where("user_id = $1", userID).
		ToSql()

	err = pgxscan.Select(context.Background(), DB, &subs, sql, args...)
	if err != nil {
		return nil, err
	}
	return subs, nil
}
func DeleteSubdiscepto(name string) error {
	sql, args, _ := psql.
		Delete("subdisceptos").
		Where("name = $1", name).
		ToSql()

	_, err := DB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func CreateReport(report *models.Report) error {
	sql, args, _ := psql.
		Insert("reports").
		Columns("flag", "essay_id", "from_user_id", "to_user_id").
		Values(report.Flag, report.EssayID, report.FromUserID, report.ToUserID).
		Suffix("RETURNING id").
		ToSql()
	row := DB.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&report.ID)
	if err != nil {
		return err
	}
	return nil
}

//TODO: Add on delete cascade to schema
func DeleteReport(report *models.Report) error {
	sql, args, _ := psql.
		Delete("reports").
		Where("id = $1", report.ID).
		ToSql()

	_, err := DB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func CreateVote(vote *models.Vote) error {
	sql, args, _ := psql.
		Insert("votes").
		Columns("user_id", "essay_id", "vote_type").
		Values(vote.UserID, vote.EssayID, vote.VoteType).
		ToSql()

	_, err := DB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
func CountVotes(essayID int) (upvotes, downvotes int, err error) {
	sql, args, _ := psql.
		Select("COUNT(*), SUM(CASE WHEN vote_type = 'upvote' THEN 1 ELSE 0 END) AS upvotes").
		From("votes").
		Where("essay_id = $1", essayID).
		GroupBy("vote_type").
		ToSql()

	row := DB.QueryRow(context.Background(), sql, args...)
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
func DeleteVote(essayID, userID int) error {
	sql, args, _ := psql.
		Delete("votes").
		Where("user_id = $1 AND essay_id = $2", userID, essayID).
		ToSql()
	_, err := DB.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	return nil
}
