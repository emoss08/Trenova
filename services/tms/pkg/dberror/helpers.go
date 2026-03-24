package dberror

import (
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

func CreateVersionMismatchError(entityName, entityID string) error {
	emitConcurrencyEvent(ConcurrencyEvent{
		Kind:   "version_mismatch",
		Entity: entityName,
	})

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

func CreateBulkVersionMismatchError(entityName string, entityIDs []pulid.ID) error {
	emitConcurrencyEvent(ConcurrencyEvent{
		Kind:   "version_mismatch",
		Entity: entityName,
	})

	return errortypes.NewValidationError(
		"version",
		errortypes.ErrVersionMismatch,
		fmt.Sprintf(
			"Version mismatch. The %s (%s) have either been updated or deleted since the last request.",
			entityName,
			strings.Join(
				pulid.Map(entityIDs, func(id pulid.ID) string { return id.String() }),
				", ",
			),
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

func IsRetryableTransactionError(err error) bool {
	code := ExtractCode(err)
	return code == pgerrcode.SerializationFailure ||
		code == pgerrcode.DeadlockDetected ||
		code == pgerrcode.LockNotAvailable
}

func NewConcurrentAccessError(message string, err error) error {
	return errortypes.NewConflictError(message).WithInternal(err)
}

func MapRetryableTransactionError(err error, message string) error {
	if !IsRetryableTransactionError(err) {
		return err
	}

	emitConcurrencyEvent(ConcurrencyEvent{
		Kind: "retryable_transaction",
		Code: ExtractCode(err),
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
}

func extractPostgresErrorDetails(err error) (postgresErrorDetails, bool) {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return postgresErrorDetails{
			code:       pgErr.Code,
			constraint: pgErr.ConstraintName,
		}, true
	}

	var pgDriverErr pgdriver.Error
	if errors.As(err, &pgDriverErr) {
		return postgresErrorDetails{
			code:       pgDriverErr.Field('C'),
			constraint: pgDriverErr.Field('n'),
		}, true
	}

	return postgresErrorDetails{}, false
}
