package common

import (
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rotisserie/eris"
)

// HandleNotFoundError converts sql.ErrNoRows to domain-specific error
func HandleNotFoundError(err error, entityName string) error {
	if eris.Is(err, sql.ErrNoRows) {
		return errors.NewNotFoundError(
			fmt.Sprintf("%s not found within your organization", entityName),
		)
	}
	return err
}

// CreateVersionMismatchError creates a standardized version mismatch error
func CreateVersionMismatchError(entityName string, entityID string) error {
	return errors.NewValidationError(
		"version",
		errors.ErrVersionMismatch,
		fmt.Sprintf(
			"Version mismatch. The %s (%s) has either been updated or deleted since the last request.",
			entityName,
			entityID,
		),
	)
}

// CheckRowsAffected validates that an operation affected the expected rows
func CheckRowsAffected(result sql.Result, entityName string, entityID string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return eris.Wrap(err, "get rows affected")
	}

	if rows == 0 {
		return CreateVersionMismatchError(entityName, entityID)
	}

	return nil
}

// IsUniqueConstraintViolation checks if error is a unique constraint violation
func IsUniqueConstraintViolation(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505"
	}
	return false
}

// ExtractConstraintName extracts the constraint name from a postgres error
func ExtractConstraintName(err error) string {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.ConstraintName
	}
	return ""
}

// WrapDatabaseError wraps database errors with operation context
func WrapDatabaseError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return eris.Wrapf(err, "%s", operation)
}
