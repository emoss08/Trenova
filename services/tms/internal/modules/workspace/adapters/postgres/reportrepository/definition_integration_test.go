//go:build integration

package reportrepository

import (
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestListDefinitionsPagesOnlyWhatTheViewerMaySee(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	data := seedtest.SeedFullTestData(t, ctx, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	colleague := seedtest.NewUser(data.Organization.ID, data.BusinessUnit.ID).
		WithUsername("defs_colleague").
		WithEmail("definitions_colleague@example.com").
		WithPassword("password123").
		Build(t, ctx, tx)
	require.NoError(t, tx.Commit())

	definitions := NewDefinitionRepository(DefinitionParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	viewer := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	peer := pagination.TenantInfo{OrgID: viewer.OrgID, BuID: viewer.BuID, UserID: colleague.ID}

	create := func(
		owner pagination.TenantInfo,
		name string,
		visibility report.Visibility,
	) pulid.ID {
		created, createErr := definitions.Create(ctx, &report.ReportDefinition{
			BusinessUnitID: owner.BuID,
			OrganizationID: owner.OrgID,
			Name:           name,
			Kind:           report.DefinitionKindCustom,
			OwnerID:        owner.UserID,
			Visibility:     visibility,
			Status:         report.DefinitionStatusActive,
			CatalogVersion: "integration-catalog",
			Definition: &ir.Definition{
				Version: ir.Version,
				Entity:  "shipment",
				Columns: []ir.Column{{ID: "pro", Dimension: &ir.Dimension{Field: "proNumber"}}},
			},
			DefaultFormat: report.FormatCSV,
		}, owner.UserID)
		require.NoError(t, createErr)
		return created.ID
	}

	visible := []pulid.ID{
		create(viewer, "Viewer's private", report.VisibilityPrivate),
		create(peer, "Colleague's shared", report.VisibilityShared),
	}
	for _, name := range []string{"Private A", "Private B", "Private C"} {
		create(peer, name, report.VisibilityPrivate)
	}
	visible = append(visible, create(peer, "Colleague's second shared", report.VisibilityShared))

	list := func(limit, offset int) []pulid.ID {
		listed, listErr := definitions.List(ctx, &reportports.ListReportDefinitionsRequest{
			TenantInfo: viewer,
			ViewerID:   viewer.UserID,
			Limit:      limit,
			Offset:     offset,
		})
		require.NoError(t, listErr)
		ids := make([]pulid.ID, 0, len(listed))
		for _, definition := range listed {
			ids = append(ids, definition.ID)
		}
		return ids
	}

	first := list(2, 0)
	second := list(2, 2)
	assert.Len(t, first, 2, "a page is full when enough visible reports exist")
	assert.Len(t, second, 1)
	assert.ElementsMatch(t, visible, append(first, second...))
	assert.ElementsMatch(t, visible, list(0, 0))
}
