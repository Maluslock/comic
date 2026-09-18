package service

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidReference = errors.New("referenced record not found")
	ErrSessionNotFound  = errors.New("chat session not found")
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
