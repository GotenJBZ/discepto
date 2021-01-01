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

func CreateUser(user *models.User, passwd string) (token string, err error) {
	// Check email format
	if !utils.ValidateEmail(user.Email) {
		return "", ErrBadEmailSyntax
	}

	// Check if email is already used
	var exists bool
	err = pgxscan.Get(context.Background(),
		DB,
		&exists,
		"SELECT exists(SELECT 1 FROM users WHERE email = $1)",
		user.Email)

	if err != nil {
		return "", err
	}
	if exists {
		return "", ErrEmailAlreadyUsed
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
		return "", err
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
		return "", err
	}

	// Insert a new token
	token = utils.GenToken(TokenLen)
	sql, args, _ = psql.
		Insert("tokens").
		Columns("user_id", "token").
		Values(user.ID, token).
		ToSql()

	_, err = tx.Exec(context.Background(), sql, args...)
	if err != nil {
		return "", err
	}

	// Commit changes
	err = tx.Commit(context.Background())
	if err != nil {
		return "", err
	}
	return token, nil
}
func CheckPasswd(user *models.User, passwd string) (bool, error) {
	var hash string
	err := pgxscan.Get(
		context.Background(),
		DB,
		&hash,
		"SELECT hash FROM credentials WHERE user_id = $1",
		user.ID,
	)
	compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(passwd))
	if err != nil || compareErr != nil {
		return false, err
	}
	return true, nil
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
func DeleteUser(id int) error {
	sql, args, _ := psql.Delete("users").Where("id = $1", id).ToSql()
	_, err := DB.Exec(context.Background(), sql, args...)
	return err
}
func ListEssays() ([]models.Essay, error) {
	var essays []models.Essay
	err := pgxscan.Select(context.Background(), DB, &essays, "SELECT * FROM essays")
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
	sql, args, _ := psql.
		Insert("essays").
		Columns("thesis", "content", "attributed_to_id", "published").
		Suffix("RETURNING id").
		Values(essay.Thesis, essay.Content, essay.AttributedToID, essay.Published).
		ToSql()

	row := tx.QueryRow(context.Background(), sql, args...)
	err = row.Scan(&essay.ID)
	if err != nil {
		return fmt.Errorf("Error inserting essay in db: %w", err)
	}
	if len(essay.Tags) > LimitMaxTags {
		return ErrTooManyTags
	}

	for _, tag := range essay.Tags {
		fmt.Println(tag)
		sql, args, _ = psql.
			Insert("essay_tags").
			Columns("essay_id", "tag").
			Values(essay.ID, tag).
			ToSql()
		_, err := tx.Exec(context.Background(),
			sql, args...)
		if err != nil {
			return fmt.Errorf("Error inserting essay_tag in db: %w", err)
		}
	}
	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}
	return nil
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
func CountVotes(essayID int, vote_type models.VoteType) (int, error) {
	sql, args, _ := psql.
		Select("count(*)").
		From("votes").
		Where("essay_id = $1 AND vote_type = $2", essayID, vote_type).
		ToSql()

	row := DB.QueryRow(context.Background(), sql, args...)
	count := 0
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
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
