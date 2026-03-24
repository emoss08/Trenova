package dberror

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockResult struct {
	rowsAffected int64
	err          error
}

func (m *mockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, m.err
}

func (m *mockResult) LastInsertId() (int64, error) {
	return 0, nil
}

func TestHandleNotFoundError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		entityName string
		wantNFE    bool
		wantMsg    string
	}{
		{
			name:       "converts sql.ErrNoRows to NotFoundError",
			err:        sql.ErrNoRows,
			entityName: "Shipment",
			wantNFE:    true,
			wantMsg:    "Shipment not found within your organization",
		},
		{
			name:       "passes through other errors unchanged",
			err:        errors.New("connection refused"),
			entityName: "Shipment",
			wantNFE:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := HandleNotFoundError(tt.err, tt.entityName)
			require.Error(t, result)

			if tt.wantNFE {
				var nfe *errortypes.NotFoundError
				require.True(t, errors.As(result, &nfe))
				assert.Equal(t, tt.wantMsg, nfe.Error())
			} else {
				var nfe *errortypes.NotFoundError
				assert.False(t, errors.As(result, &nfe))
				assert.Equal(t, tt.err, result)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "returns true for sql.ErrNoRows",
			err:  sql.ErrNoRows,
			want: true,
		},
		{
			name: "returns true for wrapped sql.ErrNoRows",
			err:  fmt.Errorf("wrapped: %w", sql.ErrNoRows),
			want: true,
		},
		{
			name: "returns false for other errors",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "returns false for nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsNotFoundError(tt.err))
		})
	}
}

func TestCreateVersionMismatchError(t *testing.T) {
	t.Parallel()

	result := CreateVersionMismatchError("Shipment", "ship_123")
	require.Error(t, result)

	var ve *errortypes.Error
	require.True(t, errors.As(result, &ve))
	assert.Equal(t, "version", ve.Field)
	assert.Equal(t, errortypes.ErrVersionMismatch, ve.Code)
	assert.Contains(t, ve.Message, "Shipment")
	assert.Contains(t, ve.Message, "ship_123")
}

func TestCreateVersionMismatchErrorEmitsConcurrencyEvent(t *testing.T) {
	var events []ConcurrencyEvent

	SetConcurrencyObserver(func(event ConcurrencyEvent) {
		events = append(events, event)
	})
	t.Cleanup(func() { SetConcurrencyObserver(nil) })

	_ = CreateVersionMismatchError("Shipment", "ship_123")

	require.Len(t, events, 1)
	assert.Equal(t, ConcurrencyEvent{
		Kind:   "version_mismatch",
		Entity: "Shipment",
		Code:   "",
	}, events[0])
}

func TestCreateBulkVersionMismatchError(t *testing.T) {
	t.Parallel()

	ids := []pulid.ID{"id_001", "id_002", "id_003"}
	result := CreateBulkVersionMismatchError("Order", ids)
	require.Error(t, result)

	var ve *errortypes.Error
	require.True(t, errors.As(result, &ve))
	assert.Equal(t, "version", ve.Field)
	assert.Equal(t, errortypes.ErrVersionMismatch, ve.Code)
	assert.Contains(t, ve.Message, "Order")
	assert.Contains(t, ve.Message, "id_001")
	assert.Contains(t, ve.Message, "id_002")
	assert.Contains(t, ve.Message, "id_003")
}

func TestCreateBulkVersionMismatchErrorEmitsConcurrencyEvent(t *testing.T) {
	var events []ConcurrencyEvent

	SetConcurrencyObserver(func(event ConcurrencyEvent) {
		events = append(events, event)
	})
	t.Cleanup(func() { SetConcurrencyObserver(nil) })

	_ = CreateBulkVersionMismatchError("Order", []pulid.ID{"id_001"})

	require.Len(t, events, 1)
	assert.Equal(t, ConcurrencyEvent{
		Kind:   "version_mismatch",
		Entity: "Order",
		Code:   "",
	}, events[0])
}

func TestCheckRowsAffected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		result       sql.Result
		entityName   string
		entityID     string
		wantErr      bool
		wantMismatch bool
	}{
		{
			name:         "returns nil when rows affected > 0",
			result:       &mockResult{rowsAffected: 1},
			entityName:   "Shipment",
			entityID:     "ship_123",
			wantErr:      false,
			wantMismatch: false,
		},
		{
			name:         "returns version mismatch when rows affected is 0",
			result:       &mockResult{rowsAffected: 0},
			entityName:   "Shipment",
			entityID:     "ship_123",
			wantErr:      true,
			wantMismatch: true,
		},
		{
			name:         "returns error when RowsAffected fails",
			result:       &mockResult{err: errors.New("driver error")},
			entityName:   "Shipment",
			entityID:     "ship_123",
			wantErr:      true,
			wantMismatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CheckRowsAffected(tt.result, tt.entityName, tt.entityID)

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)

			if tt.wantMismatch {
				var ve *errortypes.Error
				require.True(t, errors.As(err, &ve))
				assert.Equal(t, errortypes.ErrVersionMismatch, ve.Code)
			}
		})
	}
}

func TestCheckBulkRowsAffected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		result       sql.Result
		entityName   string
		entityIDs    []pulid.ID
		wantErr      bool
		wantMismatch bool
	}{
		{
			name:         "returns nil when rows affected > 0",
			result:       &mockResult{rowsAffected: 3},
			entityName:   "Order",
			entityIDs:    []pulid.ID{"id_001", "id_002", "id_003"},
			wantErr:      false,
			wantMismatch: false,
		},
		{
			name:         "returns version mismatch when rows affected is 0",
			result:       &mockResult{rowsAffected: 0},
			entityName:   "Order",
			entityIDs:    []pulid.ID{"id_001", "id_002"},
			wantErr:      true,
			wantMismatch: true,
		},
		{
			name:         "returns error when RowsAffected fails",
			result:       &mockResult{err: errors.New("driver error")},
			entityName:   "Order",
			entityIDs:    []pulid.ID{"id_001"},
			wantErr:      true,
			wantMismatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CheckBulkRowsAffected(tt.result, tt.entityName, tt.entityIDs)

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)

			if tt.wantMismatch {
				var ve *errortypes.Error
				require.True(t, errors.As(err, &ve))
				assert.Equal(t, errortypes.ErrVersionMismatch, ve.Code)
				assert.Contains(t, ve.Message, "id_001")
			}
		})
	}
}

func TestIsUniqueConstraintViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "returns true for unique violation code",
			err:  &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "unique_email"},
			want: true,
		},
		{
			name: "returns false for other pg error codes",
			err:  &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation, ConstraintName: "fk_user"},
			want: false,
		},
		{
			name: "returns false for non-pg errors",
			err:  errors.New("generic error"),
			want: false,
		},
		{
			name: "returns false for nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsUniqueConstraintViolation(tt.err))
		})
	}
}

func TestIsConstraintViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unique violation", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}, want: true},
		{name: "foreign key violation", err: &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}, want: true},
		{name: "non-constraint violation", err: &pgconn.PgError{Code: pgerrcode.SerializationFailure}, want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsConstraintViolation(tt.err))
		})
	}
}

func TestConstraintSpecificHelpers(t *testing.T) {
	t.Parallel()

	assert.True(t, IsForeignKeyConstraintViolation(&pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}))
	assert.False(t, IsForeignKeyConstraintViolation(&pgconn.PgError{Code: pgerrcode.UniqueViolation}))

	assert.True(t, IsNotNullConstraintViolation(&pgconn.PgError{Code: pgerrcode.NotNullViolation}))
	assert.False(t, IsNotNullConstraintViolation(&pgconn.PgError{Code: pgerrcode.UniqueViolation}))

	assert.True(t, IsCheckConstraintViolation(&pgconn.PgError{Code: pgerrcode.CheckViolation}))
	assert.False(t, IsCheckConstraintViolation(&pgconn.PgError{Code: pgerrcode.UniqueViolation}))
}

func TestIsRetryableTransactionError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "serialization failure", err: &pgconn.PgError{Code: pgerrcode.SerializationFailure}, want: true},
		{name: "deadlock detected", err: &pgconn.PgError{Code: pgerrcode.DeadlockDetected}, want: true},
		{name: "lock not available", err: &pgconn.PgError{Code: pgerrcode.LockNotAvailable}, want: true},
		{name: "other pg error", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}, want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsRetryableTransactionError(tt.err))
		})
	}
}

func TestMapRetryableTransactionError(t *testing.T) {
	t.Parallel()

	t.Run("maps retryable postgres errors to conflict errors", func(t *testing.T) {
		t.Parallel()

		err := MapRetryableTransactionError(&pgconn.PgError{Code: pgerrcode.LockNotAvailable}, "busy")
		require.Error(t, err)

		var conflictErr *errortypes.ConflictError
		require.True(t, errors.As(err, &conflictErr))
		assert.Equal(t, "busy", conflictErr.Message)
		assert.NotNil(t, conflictErr.Internal)
	})

	t.Run("passes through non retryable errors", func(t *testing.T) {
		t.Parallel()

		original := errors.New("plain")
		assert.Same(t, original, MapRetryableTransactionError(original, "ignored"))
	})
}

func TestMapRetryableTransactionErrorEmitsConcurrencyEvent(t *testing.T) {
	var events []ConcurrencyEvent

	SetConcurrencyObserver(func(event ConcurrencyEvent) {
		events = append(events, event)
	})
	t.Cleanup(func() { SetConcurrencyObserver(nil) })

	err := MapRetryableTransactionError(&pgconn.PgError{Code: pgerrcode.LockNotAvailable}, "busy")
	require.Error(t, err)

	require.Len(t, events, 1)
	assert.Equal(t, ConcurrencyEvent{
		Kind:   "retryable_transaction",
		Entity: "",
		Code:   pgerrcode.LockNotAvailable,
	}, events[0])
}

func TestExtractConstraintName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "extracts constraint name from PgError",
			err:  &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "unique_name"},
			want: "unique_name",
		},
		{
			name: "returns empty string for non-pg error",
			err:  errors.New("generic error"),
			want: "",
		},
		{
			name: "returns empty string for nil",
			err:  nil,
			want: "",
		},
		{
			name: "returns empty constraint name when not set",
			err:  &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ExtractConstraintName(tt.err))
		})
	}
}

func TestExtractCode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, pgerrcode.UniqueViolation, ExtractCode(&pgconn.PgError{Code: pgerrcode.UniqueViolation}))
	assert.Equal(t, "", ExtractCode(errors.New("generic error")))
	assert.Equal(t, "", ExtractCode(nil))
}

func TestExtractCodeName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "UniqueViolation", ExtractCodeName(&pgconn.PgError{Code: pgerrcode.UniqueViolation}))
	assert.Equal(t, "", ExtractCodeName(errors.New("generic error")))
}
