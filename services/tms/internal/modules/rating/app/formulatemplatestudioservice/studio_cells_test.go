package formulatemplatestudioservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/formulatemplatetest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStudioCells_RefusesNeedsItWillNotServe(t *testing.T) {
	t.Parallel()

	tooMany := make([]rules.TariffNeed, MaxStudioCellNeeds+1)
	for i := range tooMany {
		tooMany[i] = rules.TariffNeed{Table: "Grid", Keys: []string{"A"}}
	}
	tests := []struct {
		name  string
		needs []rules.TariffNeed
	}{
		{name: "no needs", needs: nil},
		{name: "too many needs", needs: tooMany},
		{name: "a need without a table", needs: []rules.TariffNeed{{Keys: []string{"A"}}}},
		{
			name:  "a need with more keys than a table has axes",
			needs: []rules.TariffNeed{{Table: "Grid", Keys: []string{"A", "B", "C", "D", "E"}}},
		},
		{
			name:  "a key longer than any stored key",
			needs: []rules.TariffNeed{{Table: "Grid", Keys: []string{strings.Repeat("k", 201)}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps := setupTest(t)
			_, err := deps.svc.StudioCells(t.Context(), &StudioCellsRequest{
				TenantInfo: formulatemplatetest.NewTenantInfo(),
				Needs:      tt.needs,
			})
			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
			assert.Equal(t, fieldNeeds, validation.Field)
		})
	}
}

func TestStudioCells_AnswersTheTablesStampAndLeavesOutUnknownTables(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	cells, err := deps.svc.StudioCells(t.Context(), &StudioCellsRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Needs:      []rules.TariffNeed{{Table: "Missing", Keys: []string{"A"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, formulatemplatetest.TablesStamp, cells.Stamp)
	assert.Empty(t, cells.Tables)
}

func TestStudioContext_CarriesTheTablesStamp(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	studio, err := deps.svc.StudioContext(t.Context(), formulatemplatetest.NewTenantInfo())
	require.NoError(t, err)
	assert.Equal(t, formulatemplatetest.TablesStamp, studio.TablesStamp)
	require.NotNil(t, studio.Context)
}
