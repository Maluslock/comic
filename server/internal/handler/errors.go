package handler

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// isNotFound reports whether err is a pgx "no rows" error, indicating
// the requested entity could not be found in the database.
func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
