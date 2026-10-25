package dberror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const parityID = "com_1"

var parityMapping = Mapping{
	Entity: "Commodity",
	Unique: []Constraint{
		{Name: "idx_commodities_name", Field: "name", Message: "Commodity with this name already exists"},
	},
	ForeignKey: []Constraint{
		{Name: "fk_commodities_hazmat", Field: "hazardousMaterialId", Message: "Hazardous material does not exist"},
	},
}

func pgError(code, constraint string) *pgconn.PgError {
	return &pgconn.PgError{Code: code, ConstraintName: constraint, Message: "violation"}
}

func TestMapTurnsKeelAndPostgresErrorsIntoTheCallersErrors(t *testing.T) {
	t.Parallel()

	unique := pgError(pgerrcode.UniqueViolation, "idx_commodities_name")
	foreignKey := pgError(pgerrcode.ForeignKeyViolation, "fk_commodities_hazmat")

	tests := []struct {
		name string
		keel error
		want error
	}{
		{
			name: "not found carries the entity name",
			keel: fmt.Errorf("commodities: %w", keel.ErrNotFound),
			want: errortypes.NewNotFoundError(
				"{0} not found within your organization", "Commodity",
			),
		},
		{
			name: "optimistic lock conflict is a version mismatch on the entity",
			keel: &keel.ConflictError{Table: "commodities", Version: 3},
			want: CreateVersionMismatchError(parityMapping.Entity, parityID),
		},
		{
			name: "unique violation names the request field",
			keel: unique,
			want: errortypes.NewValidationError(
				"name", errortypes.ErrDuplicate, "Commodity with this name already exists",
			),
		},
		{
			name: "foreign key violation names the request field",
			keel: foreignKey,
			want: errortypes.NewValidationError(
				"hazardousMaterialId", errortypes.ErrInvalidReference, "Hazardous material does not exist",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, parityMapping.Map(tt.keel, parityID))
		})
	}
}

func TestMapLeavesUnmappedErrorsAlone(t *testing.T) {
	t.Parallel()

	other := pgError(pgerrcode.UniqueViolation, "idx_commodities_code")
	assert.Same(t, other, parityMapping.Map(other, parityID))

	plain := errors.New("connection refused")
	require.ErrorIs(t, parityMapping.Map(plain, parityID), plain)

	assert.NoError(t, parityMapping.Map(nil, parityID))
}

func TestKeelErrorsAnswerTheExistingPredicates(t *testing.T) {
	t.Parallel()

	assert.True(t, IsNotFoundError(fmt.Errorf("shipments: %w", keel.ErrNotFound)))
	assert.True(t, IsVersionConflict(&keel.ConflictError{Table: "shipments", Version: 1}))
	assert.True(t, IsReadOnlyTransaction(fmt.Errorf("shipments: %w", keel.ErrReadOnly)))
	assert.True(t, IsTenantBoundaryViolation(fmt.Errorf("shipments: %w", keel.ErrTenantMismatch)))
	assert.True(t, IsTenantBoundaryViolation(&pgconn.PgError{
		Code:    pgerrcode.InsufficientPrivilege,
		Message: "new row violates row-level security policy for table \"shipments\"",
	}))
	assert.False(t, IsTenantBoundaryViolation(keel.ErrNoScope))
}

func TestHandleVersionConflict(t *testing.T) {
	t.Parallel()

	mapped := HandleVersionConflict(
		fmt.Errorf("save: %w", &keel.ConflictError{Table: "shipments", Version: 2}),
		"Shipment",
		"shp_1",
	)
	assert.Equal(t, CreateVersionMismatchError("Shipment", "shp_1"), mapped)

	other := errors.New("boom")
	assert.Same(t, other, HandleVersionConflict(other, "Shipment", "shp_1"))
}

func TestCountCheckers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, CreateVersionMismatchError("Shipment", "shp_1"), CheckAffected(0, "Shipment", "shp_1"))
	require.Error(t, CheckFoundCount(0, "Shipment"))
	assert.NoError(t, CheckAffected(1, "Shipment", "shp_1"))
	assert.NoError(t, CheckFoundCount(1, "Shipment"))
	assert.NoError(t, CheckBulkAffected(2, "Shipment", nil))
}
