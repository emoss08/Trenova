package dberror

import (
	"errors"
	"strings"

	"github.com/jackc/pgerrcode"
	sqlitedriver "modernc.org/sqlite"
)

const (
	sqliteBusy              = 5
	sqliteLocked            = 6
	sqliteInterrupt         = 9
	sqliteConstraintCheck   = 275
	sqliteConstraintFK      = 787
	sqliteConstraintUnique  = 2067
	sqliteConstraintPrimary = 1555
	sqliteConstraintNotNull = 1299
	sqliteBusySnapshot      = 517
	sqliteBusyTimeout       = 261
	sqliteLockedSharedCache = 262
)

var sqliteToSQLState = map[int]string{
	sqliteConstraintUnique:  pgerrcode.UniqueViolation,
	sqliteConstraintPrimary: pgerrcode.UniqueViolation,
	sqliteConstraintFK:      pgerrcode.ForeignKeyViolation,
	sqliteConstraintNotNull: pgerrcode.NotNullViolation,
	sqliteConstraintCheck:   pgerrcode.CheckViolation,
	sqliteBusy:              pgerrcode.SerializationFailure,
	sqliteBusySnapshot:      pgerrcode.SerializationFailure,
	sqliteBusyTimeout:       pgerrcode.SerializationFailure,
	sqliteLocked:            pgerrcode.LockNotAvailable,
	sqliteLockedSharedCache: pgerrcode.LockNotAvailable,
	sqliteInterrupt:         pgerrcode.QueryCanceled,
}

func extractSQLiteErrorDetails(err error) (postgresErrorDetails, bool) {
	sqliteErr, ok := errors.AsType[*sqlitedriver.Error](err)
	if !ok {
		return postgresErrorDetails{}, false
	}

	code, mapped := sqliteToSQLState[sqliteErr.Code()]
	if !mapped {
		code = sqliteBaseCodeToSQLState(sqliteErr.Code())
	}

	return postgresErrorDetails{
		code:       code,
		constraint: parseSQLiteConstraint(sqliteErr.Error()),
	}, true
}

func sqliteBaseCodeToSQLState(code int) string {
	switch code & 0xff {
	case sqliteBusy, sqliteLocked:
		return pgerrcode.SerializationFailure
	case 19:
		return pgerrcode.IntegrityConstraintViolation
	default:
		return ""
	}
}

func parseSQLiteConstraint(message string) string {
	const (
		uniquePrefix  = "UNIQUE constraint failed: "
		primaryPrefix = "PRIMARY KEY constraint failed: "
		checkPrefix   = "CHECK constraint failed: "
		notNullPrefix = "NOT NULL constraint failed: "
	)

	for _, prefix := range []string{uniquePrefix, primaryPrefix, checkPrefix, notNullPrefix} {
		idx := strings.Index(message, prefix)
		if idx < 0 {
			continue
		}

		target := strings.TrimSpace(message[idx+len(prefix):])
		if end := strings.IndexAny(target, "\n("); end >= 0 {
			target = strings.TrimSpace(target[:end])
		}

		return target
	}

	return ""
}
