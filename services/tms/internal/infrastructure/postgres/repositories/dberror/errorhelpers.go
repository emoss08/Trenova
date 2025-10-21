package dberror

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/jackc/pgconn"
)

// HandleNotFoundError converts sql.ErrNoRows to domain-specific error
func HandleNotFoundError(err error, entityName string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return errortypes.NewNotFoundError(
			fmt.Sprintf("%s not found within your organization", entityName),
		)
	}

	return err
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// CreateVersionMismatchError creates a standardized version mismatch error
func CreateVersionMismatchError(entityName, entityID string) error {
	return errortypes.NewValidationError(
		"version",
		errortypes.ErrVersionMismatch,
		fmt.Sprintf(
			"Version mismatch. The %s (%s) has either been updated or deleted since the last request.",
			entityName,
			entityID,
		),
	)
}

// CheckRowsAffected validates that an operation affected the expected rows
func CheckRowsAffected(result sql.Result, entityName, entityID string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return CreateVersionMismatchError(entityName, entityID)
	}

	return nil
}

// IsUniqueConstraintViolation checks if error is a unique constraint violation
func IsUniqueConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// ExtractConstraintName extracts the constraint name from a postgres error
func ExtractConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// WrapDatabaseError wraps database errors with operation context
func WrapDatabaseError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
