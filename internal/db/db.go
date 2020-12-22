package db

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"gitlab.com/ranfdev/discepto/internal/models"
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

func ListUsers() ([]models.User, error) {
	var users []models.User
	err := pgxscan.Select(context.Background(), DB, &users, "SELECT * FROM users")
	return users, err
}

func CreateUser(user *models.User) error {
	_, err := DB.Exec(context.Background(), "INSERT INTO users (name, email, role_id) VALUES ($1, $2, $3)", user.Name, user.Email, user.RoleID)
	return err
}
