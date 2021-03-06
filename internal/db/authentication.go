package db

import (
	"context"
	"errors"
	"unicode"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

func (sdb *SharedDB) CreateUser(ctx context.Context, user *models.User, passwd string) (uH *UserH, err error) {
	// Check email format
	if !utils.ValidateEmail(user.Email) {
		return nil, models.ErrInvalidFormat
	}

	if !validatePasswd(passwd, []string{user.Email, user.Name}) {
		return nil, models.ErrWeakPasswd
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(passwd), sdb.bcryptCost)

	err = execTx(ctx, sdb.db, func(ctx context.Context, tx DBTX) error {
		err = insertUser(ctx, sdb.db, user, hash)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "users_email_key" {
			return models.ErrEmailAlreadyUsed
		} else if err != nil {
			return err
		}

		// Assign admin role if first user
		sql, args, _ := psql.Select("COUNT(*)").From("users").ToSql()
		c := 0
		row := tx.QueryRow(ctx, sql, args...)
		err = row.Scan(&c)
		if err != nil {
			return err
		}

		if c == 1 {
			err := assignRole(ctx, tx, user.ID, RoleDisceptoAdmin.ID)
			if err != nil {
				return err
			}
		}
		err := assignRole(ctx, tx, user.ID, RoleDisceptoCommon.ID)
		if err != nil {
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
func (sdb *SharedDB) Login(ctx context.Context, email string, passwd string) (*UserH, error) {
	sql, args, _ := psql.
		Select("id", "passwd_hash").
		From("users").
		Where(sq.Eq{"email": email}).
		ToSql()

	var data struct {
		ID         int
		PasswdHash string
	}
	err := pgxscan.Get(
		ctx,
		sdb.db,
		&data,
		sql,
		args...,
	)
	if err != nil {
		return nil, err
	}
	compareErr := bcrypt.CompareHashAndPassword([]byte(data.PasswdHash), []byte(passwd))
	if compareErr != nil {
		return nil, compareErr
	}

	return &UserH{
		id: data.ID,
		perms: userPerms{
			Read:   true,
			Delete: true,
		},
		sharedDB: sdb.db,
	}, nil
}
func validatePasswd(passwd string, userInputs []string) bool {
	if len(passwd) < 8 || len(passwd) > 64 {
		return false
	}

	containsLetter := false
	containsNumber := false
	containsSpecial := false
	for _, r := range passwd {
		if !unicode.IsPrint(r) {
			return false
		}

		if unicode.IsLetter(r) {
			containsLetter = true
		} else if unicode.IsNumber(r) {
			containsNumber = true
		} else {
			// If it's not a number and not a letter, it's special
			containsSpecial = true
		}
	}
	if !containsLetter || !containsNumber || !containsSpecial {
		return false
	}

	return true
}
func insertUser(ctx context.Context, db DBTX, user *models.User, hash []byte) error {
	// Insert the new user
	sql, args, _ := psql.
		Insert("users").
		Columns("name", "email", "passwd_hash").
		Values(user.Name, user.Email, hash).
		Suffix("RETURNING id").
		ToSql()

	row := db.QueryRow(ctx, sql, args...)
	err := row.Scan(&user.ID)
	return err
}
