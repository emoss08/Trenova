package carrierservice

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestValidatorUniquenessChecksUseTableColumns(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)
	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() { bunDB.Close() })

	for _, column := range []string{"code", "dot_number", "scac"} {
		mock.ExpectQuery(`LOWER\(carriers\.` + column + `\) = LOWER\(`).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	}

	validator := NewValidator(ValidatorParams{DB: postgres.NewTestConnection(bunDB)})
	entity := &carrier.Carrier{
		ID:             pulid.MustNew("car_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Code:           "BLUE",
		Name:           "Blue Ridge Transport",
		DOTNumber:      "1204471",
		SCAC:           "BLRT",
	}

	multiErr := validator.ValidateUpdate(t.Context(), entity)
	if multiErr != nil {
		for _, e := range multiErr.Errors {
			assert.NotEqual(t, "system", e.Field, e.Message)
		}
	}
	require.NoError(t, mock.ExpectationsWereMet())
}
