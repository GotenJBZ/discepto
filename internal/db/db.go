package db

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/utils"
)

var DB *pgxpool.Pool

func Connect() error {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		return errors.New("DATABASE_URL env variable missing")
	}
	// dbUrl := "postgres://discepto:passwd@localhost/disceptoDb"
	var err error = nil
	DB, err = pgxpool.Connect(context.Background(), dbUrl)
	if err != nil {
		err = fmt.Errorf("Failed to connect to postgres with url `%s`: %w", dbUrl, err)
	}
	return err
}

func Migrate() error {
	url := os.Getenv("DATABASE_URL")
	m, err := migrate.New("file://migrations", url)
	if err != nil {
		return fmt.Errorf("Error creating migrations: %s", err)
	}
	defer m.Close()
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("While upping: %s", err)
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
		return errors.New("Email validation not passed")
	}
	_, err := DB.Exec(context.Background(), "INSERT INTO users (name, email, role_id) VALUES ($1, $2, $3)", user.Name, user.Email, user.RoleID)
	return err
}
