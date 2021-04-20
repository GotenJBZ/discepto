// This file contains public queries which doesn't require any kind
// of access control
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

func (sdb *SharedDB) ListRecentEssaysIn(ctx context.Context, subs []string) ([]models.EssayView, error) {
	essayPreviews := []models.EssayView{}
	sql, args, _ := selectEssayWithJoins.
		Where(sq.Eq{"posted_in": subs}).
		GroupBy("essays.id", "essay_replies.from_id", "users.name").
		ToSql()

	err := pgxscan.Select(ctx, sdb.db, &essayPreviews, sql, args...)
	if err != nil {
		return nil, err
	}
	return essayPreviews, nil
}
func (sdb *SharedDB) ListSubdisceptos(ctx context.Context, userH *UserH) ([]models.SubdisceptoView, error) {
	var subs []models.SubdisceptoView
	var userID *int
	if userH != nil {
		userID = &userH.id
	}
	sql, args, _ := selectSubdiscepto(userID).ToSql()
	err := pgxscan.Select(ctx, sdb.db, &subs, sql, args...)
	if err != nil {
		return nil, err
	}
	return subs, nil
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
func (sdb *SharedDB) CreateUser(ctx context.Context, user *models.User, passwd string) (uH *UserH, err error) {
	// Check email format
	if !utils.ValidateEmail(user.Email) {
		return nil, ErrInvalidFormat
	}

	if !validatePasswd(passwd, []string{user.Email, user.Name}) {
		return nil, ErrWeakPasswd
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(passwd), sdb.bcryptCost)

	err = execTx(ctx, sdb.db, func(ctx context.Context, tx DBTX) error {
		err = insertUser(ctx, sdb.db, user, hash)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "users_email_key" {
			return ErrEmailAlreadyUsed
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
			err := assignGlobalRole(ctx, tx, nil, user.ID, "admin", true)
			return err
		}
		err := assignGlobalRole(ctx, tx, nil, user.ID, "common", true)
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
func (sdb *SharedDB) Login(ctx context.Context, email string, passwd string) (token string, err error) {
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
		ctx,
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

	_, err = sdb.db.Exec(ctx, sql, args...)
	if err != nil {
		return "", err
	}
	return token, nil
}
func (sdb *SharedDB) Signout(ctx context.Context, token string) error {
	_, err := sdb.db.Exec(ctx, "DELETE FROM tokens WHERE tokens.token = $1", token)
	if err != nil {
		return err
	}
	return nil
}
func (sdb *SharedDB) searchByTags(ctx context.Context, tags []string) (essays []*models.Essay, err error) {
	sql, args, _ := psql.
		Select("thesis", "content", "reply_type").
		Distinct().
		From("essays").
		LeftJoin("essay_tags ON id = essay_id").
		Where(sq.Eq{"tag": tags}).
		ToSql()

	err = pgxscan.Select(ctx, sdb.db, &essays, sql, args...)
	if err != nil {
		return nil, err
	}
	return essays, nil
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
