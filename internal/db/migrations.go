package db

import (
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrate() error {
	url := os.Getenv("DATABASE_URL")
	m, err := migrate.New("file://migrations", url)
	if err != nil {
		return err
	}
	defer m.Close()
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("While upping: %s", err)
	}
	return nil
}
