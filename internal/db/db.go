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
	"github.com/jackc/pgconn"
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

var ErrTooManyTags error = errors.New("You have inserted too many tags")
var ErrBadContentLen error = errors.New("You have to respect the imposed content length limits")
var ErrEmailAlreadyUsed error = errors.New("The email is already used")
var ErrInvalidFormat error = errors.New("Invalid format")
var ErrPermDenied = errors.New("Not enough permissions to execute this action")

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

func Connect(config *models.EnvConfig) (SharedDB, error) {
	db, err := pgxpool.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		err = fmt.Errorf("Failed to connect to postgres: %w", err)
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
func insertUser(db DBTX, user *models.User, hash []byte) error {
	// Insert the new user
	sql, args, _ := psql.
		Insert("users").
		Columns("name", "email", "passwd_hash").
		Values(user.Name, user.Email, hash).
		Suffix("RETURNING id").
		ToSql()

	row := db.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&user.ID)
	return err
}
func (sdb *SharedDB) CreateUser(user *models.User, passwd string) (uH *UserH, err error) {
	// Check email format
	if !utils.ValidateEmail(user.Email) {
		return nil, ErrInvalidFormat
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(passwd), sdb.bcryptCost)

	err = execTx(context.Background(), sdb.db, func(ctx context.Context, tx DBTX) error {
		err = insertUser(sdb.db, user, hash)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "users_email_key" {
			return ErrEmailAlreadyUsed
		} else if err != nil {
			return err
		}

		// Assign admin role if first user
		sql, args, _ := psql.Select("COUNT(*)").From("users").ToSql()
		c := 0
		row := tx.QueryRow(context.Background(), sql, args...)
		err = row.Scan(&c)
		if err != nil {
			return err
		}

		if c == 1 {
			err := assignNamedGlobalRole(tx, user.ID, "admin", true)
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Return handle to this user
	uH = &UserH{
		id:       user.ID,
		perms:    userPerms{Delete: true, Read: true},
		sharedDB: sdb.db,
	}

	return uH, nil
}
func (sdb *SharedDB) Login(email string, passwd string) (token string, err error) {
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
		sdb.db,
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

	_, err = sdb.db.Exec(context.Background(), sql, args...)
	if err != nil {
		return "", err
	}
	return token, nil
}
func (sdb *SharedDB) Signout(token string) error {
	_, err := sdb.db.Exec(context.Background(), "DELETE FROM tokens WHERE tokens.token = $1", token)
	if err != nil {
		return err
	}
	return nil
}
func (sdb *SharedDB) searchByTags(tags []string) (essays []*models.Essay, err error) {
	sql, args, _ := psql.
		Select("thesis", "content", "reply_type").
		Distinct().
		From("essays").
		LeftJoin("essay_tags ON id = essay_id").
		Where(sq.Eq{"tag": tags}).
		ToSql()

	err = pgxscan.Select(context.Background(), sdb.db, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
}

func (sdb *SharedDB) ListRecentEssaysIn(subs []string) (essays []*models.Essay, err error) {
	sql, args, _ := psql.
		Select("*").
		From("essays").
		Where(sq.Eq{"posted_in": subs}).
		ToSql()

	err = pgxscan.Select(context.Background(), sdb.db, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
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

func bool_or(col string) string {
	return fmt.Sprintf("bool_or(%s) AS %s", col, col)
}
func assignNamedGlobalRole(tx DBTX, userID int, role string, preset bool) error {
	sql, args, _ := sq.
		Insert("user_global_roles").
		Columns("user_id", "global_perms_id", "sub_perms_id").
		Select(sq.Select(fmt.Sprint(userID), "global_perms_id", "sub_perms_id").
			From("global_roles").
			Where(sq.Eq{
				"name":   role,
				"preset": preset,
			})).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	fmt.Println(sql)
	_, err := tx.Exec(context.Background(), sql, args...)
	return err
}
func assignNamedSubRole(db DBTX, userID int, sub string, role string, preset bool) error {
	sql := `
INSERT INTO user_sub_roles (subdiscepto, user_id, sub_perms_id)
SELECT $1, $2, sub_perms_id
FROM sub_roles
WHERE (subdiscepto = $3 OR subdiscepto IS NULL) AND name = $4 AND preset = $5
`
	_, err := db.Exec(context.Background(), sql, sub, userID, sub, role, preset)
	return err
}

func (sdb *SharedDB) ListMySubdisceptos(userH UserH) (subs []string, err error) {
	sql, args, _ := psql.
		Select("subdiscepto").
		From("subdiscepto_users").
		Where(sq.Eq{"user_id": userH.id}).
		ToSql()

	err = pgxscan.Select(context.Background(), sdb.db, &subs, sql, args...)
	if err != nil {
		return nil, err
	}
	return subs, nil
}
func (sdb *SharedDB) CreateReport(report *models.Report) error {
	sql, args, _ := psql.
		Insert("reports").
		Columns("flag", "essay_id", "from_user_id").
		Values(report.Flag, report.EssayID, report.FromUserID).
		Suffix("RETURNING id").
		ToSql()
	row := sdb.db.QueryRow(context.Background(), sql, args...)
	err := row.Scan(&report.ID)
	if err != nil {
		return err
	}
	return nil
}
func (sdb *SharedDB) ListSubdisceptos() ([]*models.Subdiscepto, error) {
	var subs []*models.Subdiscepto
	err := pgxscan.Select(context.Background(), sdb.db, &subs, "SELECT name, description, min_length, questions_required, nsfw FROM subdisceptos")
	if err != nil {
		return nil, err
	}
	return subs, nil
}
