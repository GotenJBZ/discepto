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
)

const (
	LimitMaxTags       = 10
	LimitMinContentLen = 150
	LimitMaxContentLen = 5000 // 5K
)

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

var ErrBadEmailSyntax error = errors.New("Bad email syntax")
var ErrTooManyTags error = errors.New("You have inserted too many tags")
var ErrBadContentLen error = errors.New("You have to respect the imposed content length limits")
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

func CreateUser(user *models.User) error {
	if !utils.ValidateEmail(user.Email) {
		return ErrBadEmailSyntax
	}
	sql, args, _ := psql.
		Insert("users").
		Columns("name", "email", "role_id").
		Values(user.Name, user.Email, user.RoleID).
		Suffix("RETURNING id").
		ToSql()
	row := DB.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&user.ID)
	return err
}
func DeleteUser(id int) error {
	sql, args, _ := psql.Delete("users").Where("id = $1", id).ToSql()
	_, err := DB.Exec(context.Background(), sql, args...)
	return err
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
