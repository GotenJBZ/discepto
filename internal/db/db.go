// Package db implements functions accessing the database, securely,
// checking permissions
package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/migrations"
	"golang.org/x/crypto/bcrypt"
)

const (
	LimitMaxTags       = 10
	LimitMaxContentLen = 10000 // 10K
	TokenLen           = 64    // 64 bytes
	PgErrCodeDuplicate = "23505"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type SharedDB struct {
	db         DBTX
	config     *models.EnvConfig
	bcryptCost int
}

func (sdb SharedDB) withTx(tx DBTX) SharedDB {
	sdb.db = tx
	return sdb
}

func migrator(dbURL string) (*migrate.Migrate, error) {
	dbURL = strings.Replace(dbURL, "postgres", "pgx", 1)
	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("error reading migrations: %s", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	return m, err
}
func MigrateUp(dbURL string) error {
	m, err := migrator(dbURL)
	if err != nil {
		return err
	}
	defer m.Close()
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("while migrating up: %s", err)
	}
	return nil
}
func MigrateDown(dbURL string) error {
	m, err := migrator(dbURL)
	if err != nil {
		return err
	}
	defer m.Close()
	err = m.Down()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("while migrating down: %s", err)
	}
	return nil
}
func Drop(dbURL string) error {
	m, err := migrator(dbURL)
	if err != nil {
		return err
	}
	defer m.Close()
	err = m.Drop()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("while dropping: %s", err)
	}
	return nil
}

func Connect(config *models.EnvConfig) (SharedDB, error) {
	db, err := pgxpool.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		err = fmt.Errorf("failed to connect to postgres: %w", err)
	}
	bcryptCost := bcrypt.DefaultCost + 2
	if config.Debug {
		bcryptCost = bcrypt.MinCost
	}

	return SharedDB{
		db,
		config,
		bcryptCost,
	}, err
}

func execTx(ctx context.Context, db DBTX, txFunc func(context.Context, DBTX) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	err = txFunc(ctx, tx)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	err = tx.Commit(ctx)
	return err
}
