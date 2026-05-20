package documentcontentrepository

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func TestListPendingExtractionTenants_UsesQualifiedTenantGrouping(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	matcher := sqlmock.QueryMatcherFunc(func(_ string, actualSQL string) error {
		requiredFragments := []string{
			"SELECT doc.organization_id AS organization_id, doc.business_unit_id AS business_unit_id",
			"LEFT JOIN document_contents AS dc",
			"GROUP BY doc.organization_id, doc.business_unit_id",
			`ORDER BY "doc"."organization_id" ASC, "doc"."business_unit_id" ASC`,
		}
		for _, fragment := range requiredFragments {
			if !strings.Contains(actualSQL, fragment) {
				return fmt.Errorf("expected SQL to contain %q, got %s", fragment, actualSQL)
			}
		}

		if strings.Contains(actualSQL, `GROUP BY "organization_id"`) ||
			strings.Contains(actualSQL, "GROUP BY organization_id") {
			return fmt.Errorf("tenant grouping must be table-qualified, got %s", actualSQL)
		}

		if strings.Contains(actualSQL, "doc.status = $1) OR") ||
			strings.Contains(actualSQL, "doc.status = 'Active') OR") {
			return fmt.Errorf("status filter must not be ORed with eligibility filters, got %s", actualSQL)
		}

		return nil
	})
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)

	bunDB := bun.NewDB(sqlDB, pgdialect.New())
	repo := &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}
	mock.ExpectQuery("tenant discovery").
		WillReturnRows(sqlmock.NewRows([]string{
			"organization_id",
			"business_unit_id",
		}).AddRow(orgID, buID))
	mock.ExpectClose()

	tenants, err := repo.ListPendingExtractionTenants(
		context.Background(),
		&repositories.ListPendingDocumentExtractionRequest{
			OlderThan: 1779222120,
			Limit:     100,
		},
	)
	require.NoError(t, err)
	require.Equal(t, []pagination.TenantInfo{
		{
			OrgID: orgID,
			BuID:  buID,
		},
	}, tenants)
	require.NoError(t, bunDB.Close())
	require.NoError(t, mock.ExpectationsWereMet())
}
