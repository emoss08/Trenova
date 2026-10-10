package dberror

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/uptrace/bun/driver/pgdriver"
	sqlitedriver "modernc.org/sqlite"
)

var ErrCheckRowNil = errors.New("check rows affected: result is nil")

type ConcurrencyEvent struct {
	Kind   string
	Entity string
	Code   string
}

var (
	concurrencyObserverMu sync.RWMutex
	concurrencyObserver   func(ConcurrencyEvent)
)

func SetConcurrencyObserver(observer func(ConcurrencyEvent)) {
	concurrencyObserverMu.Lock()
	defer concurrencyObserverMu.Unlock()
	concurrencyObserver = observer
}

func emitConcurrencyEvent(event ConcurrencyEvent) {
	concurrencyObserverMu.RLock()
	observer := concurrencyObserver
	concurrencyObserverMu.RUnlock()

	if observer != nil {
		observer(event)
	}
}

func HandleNotFoundError(err error, entityName string) error {
	if IsNotFoundError(err) {
		return errortypes.NewNotFoundError(
			"{0} not found within your organization", entityName,
		)
	}

	return err
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func CreateVersionMismatchError(entityName, entityID string) error {
	emitConcurrencyEvent(ConcurrencyEvent{
		Kind:   "version_mismatch",
		Entity: entityName,
	})

	return errortypes.NewValidationError(
		"version",
		errortypes.ErrVersionMismatch,
		"Version mismatch. The {0} ({1}) has either been updated or deleted since the last request.",
		entityName,
		entityID,
	)
}

func CreateBulkVersionMismatchError(entityName string, entityIDs []pulid.ID) error {
	emitConcurrencyEvent(ConcurrencyEvent{
		Kind:   "version_mismatch",
		Entity: entityName,
	})

	return errortypes.NewValidationError(
		"version",
		errortypes.ErrVersionMismatch,
		"Version mismatch. The {0} ({1}) have either been updated or deleted since the last request.",
		entityName,
		strings.Join(
			pulid.Map(entityIDs, func(id pulid.ID) string { return id.String() }),
			", ",
		),
	)
}

func CheckRowsAffected(result sql.Result, entityName, entityID string) error {
	if result == nil {
		return ErrCheckRowNil
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return CreateVersionMismatchError(entityName, entityID)
	}

	return nil
}

func CheckFound(result sql.Result, entityName string) error {
	if result == nil {
		return ErrCheckRowNil
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return errortypes.NewNotFoundError("{0} not found within your organization", entityName)
	}

	return nil
}

func CheckBulkRowsAffected(result sql.Result, entityName string, entityIDs []pulid.ID) error {
	if result == nil {
		return ErrCheckRowNil
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return CreateBulkVersionMismatchError(entityName, entityIDs)
	}

	return nil
}

func IsConstraintViolation(err error) bool {
	code := ExtractCode(err)
	return pgerrcode.IsIntegrityConstraintViolation(code)
}

func IsUniqueConstraintViolation(err error) bool {
	return ExtractCode(err) == pgerrcode.UniqueViolation
}

func IsForeignKeyConstraintViolation(err error) bool {
	return ExtractCode(err) == pgerrcode.ForeignKeyViolation
}

func IsNotNullConstraintViolation(err error) bool {
	return ExtractCode(err) == pgerrcode.NotNullViolation
}

func IsCheckConstraintViolation(err error) bool {
	return ExtractCode(err) == pgerrcode.CheckViolation
}

func DriverError(err error) error {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr
	}

	var pgDriverErr pgdriver.Error
	if errors.As(err, &pgDriverErr) {
		return pgDriverErr
	}

	if sqliteErr, ok := errors.AsType[*sqlitedriver.Error](err); ok {
		return sqliteErr
	}

	return nil
}

// IsReadOnlyTransaction reports a write attempted inside a read-only
// transaction, such as a preview's snapshot.
func IsReadOnlyTransaction(err error) bool {
	return ExtractCode(err) == pgerrcode.ReadOnlySQLTransaction
}

func IsRowLevelSecurityViolation(err error) bool {
	details, ok := extractPostgresErrorDetails(err)
	if !ok {
		return false
	}

	return details.code == pgerrcode.InsufficientPrivilege &&
		strings.Contains(details.message, "row-level security")
}

func IsRetryableTransactionError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	code := ExtractCode(err)
	return code == pgerrcode.SerializationFailure ||
		code == pgerrcode.DeadlockDetected ||
		code == pgerrcode.LockNotAvailable ||
		code == pgerrcode.QueryCanceled
}

func NewConcurrentAccessError(message string, err error) error {
	return errortypes.NewConflictError(message).WithInternal(err)
}

func MapRetryableTransactionError(err error, message string) error {
	if !IsRetryableTransactionError(err) {
		return err
	}

	code := ExtractCode(err)
	if code == "" && errors.Is(err, context.DeadlineExceeded) {
		code = "context_deadline_exceeded"
	}

	emitConcurrencyEvent(ConcurrencyEvent{
		Kind: "retryable_transaction",
		Code: code,
	})

	if message == "" {
		message = "The record is busy. Retry the request."
	}

	return NewConcurrentAccessError(message, err)
}

func ExtractConstraintName(err error) string {
	details, ok := extractPostgresErrorDetails(err)
	if !ok {
		return ""
	}

	return details.constraint
}

func ExtractCode(err error) string {
	details, ok := extractPostgresErrorDetails(err)
	if !ok {
		return ""
	}

	return details.code
}

func ExtractCodeName(err error) string {
	code := ExtractCode(err)
	if code == "" {
		return ""
	}

	return pgerrcode.Name(code)
}

type postgresErrorDetails struct {
	code       string
	constraint string
	message    string
}

func extractPostgresErrorDetails(err error) (postgresErrorDetails, bool) {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return postgresErrorDetails{
			code:       pgErr.Code,
			constraint: pgErr.ConstraintName,
			message:    pgErr.Message,
		}, true
	}

	var pgDriverErr pgdriver.Error
	if errors.As(err, &pgDriverErr) {
		return postgresErrorDetails{
			code:       pgDriverErr.Field('C'),
			constraint: pgDriverErr.Field('n'),
			message:    pgDriverErr.Field('M'),
		}, true
	}

	return extractSQLiteErrorDetails(err)
}
