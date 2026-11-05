package pgstubtest_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/testutil/pgstubtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExistsAnswersFromTheListedTables(t *testing.T) {
	t.Parallel()

	checker := keelcheck.NewChecker(
		pgstubtest.NewConnection(t, "equipment_types"),
		trailerdb.TrailersSchema(),
	)

	listed, err := checker.CheckReference(t.Context(), &validationframework.ReferenceRequest{
		TableName:      "equipment_types",
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ID:             pulid.MustNew("eqt_"),
	})
	require.NoError(t, err)
	assert.True(t, listed)

	conflict, err := checker.CheckUniqueness(t.Context(), &validationframework.UniquenessRequest{
		TableName:      "trailers",
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Fields: []validationframework.FieldCheck{
			{Column: "code", Value: "TR001", CaseSensitive: true},
		},
	})
	require.NoError(t, err)
	assert.False(t, conflict)
}

func TestCountAnswersZero(t *testing.T) {
	t.Parallel()

	conn := pgstubtest.NewConnection(t)

	count, err := trailerdb.Trailers.Query().Count(t.Context(), conn.Keel(t.Context()))
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestExistsAnswersUnderASystemScope(t *testing.T) {
	t.Parallel()

	checker := keelcheck.NewChecker(pgstubtest.NewConnection(t), trailerdb.TrailersSchema())

	conflict, err := checker.CheckUniqueness(
		dbscope.WithSystem(t.Context(), "check across the business unit"),
		&validationframework.UniquenessRequest{
			TableName:      "trailers",
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			Fields: []validationframework.FieldCheck{
				{Column: "code", Value: "TR001", CaseSensitive: true},
			},
		},
	)
	require.NoError(t, err)
	assert.False(t, conflict)
}
