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

var parityMapping = Mapping{Entity: "Commodity"}

func pgError(code, constraint string) *pgconn.PgError {
	return &pgconn.PgError{Code: code, ConstraintName: constraint, Message: "violation"}
}

func violationOf(
	kind keel.ConstraintKind,
	code, constraint, message string,
	fields, labels []string,
) *keel.ConstraintError {
	return &keel.ConstraintError{
		Kind:       kind,
		Table:      "commodities",
		Constraint: constraint,
		Fields:     fields,
		Labels:     labels,
		Message:    message,
		Err:        pgError(code, constraint),
	}
}

func TestMapTurnsKeelErrorsIntoTheCallersErrors(t *testing.T) {
	t.Parallel()

	declared := violationOf(keel.ConstraintUnique, pgerrcode.UniqueViolation, "idx_commodities_name",
		"Commodity with this name already exists", []string{"name"}, []string{"Name"})
	unique := violationOf(keel.ConstraintUnique, pgerrcode.UniqueViolation, "idx_commodities_code",
		"", []string{"code"}, []string{"Code"})
	foreignKey := violationOf(keel.ConstraintForeignKey, pgerrcode.ForeignKeyViolation,
		"fk_commodities_hazmat", "", []string{"hazardousMaterialId"}, []string{"Hazardous Material"})
	notNull := violationOf(keel.ConstraintNotNull, pgerrcode.NotNullViolation, "", "",
		[]string{"name"}, []string{"Name"})
	check := violationOf(keel.ConstraintCheck, pgerrcode.CheckViolation, "ck_commodities_temp", "",
		nil, nil)
	exclusion := violationOf(keel.ConstraintExclusion, pgerrcode.ExclusionViolation,
		"ex_commodities_window", "", nil, nil)

	tests := []struct {
		name string
		keel error
		want error
	}{
		{
			name: "not found carries the entity name",
			keel: fmt.Errorf("commodities: %w", keel.ErrNotFound),
			want: errortypes.NewNotFoundError("{0} not found within your organization", "Commodity"),
		},
		{
			name: "optimistic lock conflict is a version mismatch on the entity",
			keel: &keel.ConflictError{Table: "commodities", Version: 3},
			want: CreateVersionMismatchError(parityMapping.Entity, parityID),
		},
		{
			name: "a declared message wins",
			keel: fmt.Errorf("create: %w", declared),
			want: errortypes.NewValidationError("name", errortypes.ErrDuplicate,
				"Commodity with this name already exists").WithInternal(declared),
		},
		{
			name: "unique violation without a message names the entity and the column",
			keel: unique,
			want: errortypes.NewValidationError("code", errortypes.ErrDuplicate,
				"{0} with this {1} already exists in your organization", "Commodity", "Code").
				WithInternal(unique),
		},
		{
			name: "foreign key violation names the reference",
			keel: foreignKey,
			want: errortypes.NewValidationError("hazardousMaterialId", errortypes.ErrInvalidReference,
				"{0} does not exist in your organization", "Hazardous Material").WithInternal(foreignKey),
		},
		{
			name: "not null violation is required",
			keel: notNull,
			want: errortypes.NewValidationError("name", errortypes.ErrRequired, "{0} is required", "Name").
				WithInternal(notNull),
		},
		{
			name: "check violation without columns names the entity",
			keel: check,
			want: errortypes.NewValidationError("", errortypes.ErrInvalid, "{0} is not valid", "Commodity").
				WithInternal(check),
		},
		{
			name: "exclusion violation is an overlap",
			keel: exclusion,
			want: errortypes.NewValidationError("", errortypes.ErrDuplicate,
				"{0} overlaps an existing record in your organization", "Commodity").WithInternal(exclusion),
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

	other := pgError(pgerrcode.SerializationFailure, "")
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
