//go:build integration

package agentquerytoolservice

import (
	"fmt"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const pagedWorkers = 8

type pagedWorkerRows struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor"`
}

func TestWorkerList_PagesARealCursorRepositoryWithoutRepeats(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)

	repo := workerrepository.New(workerrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	created := make(map[string]struct{}, pagedWorkers)
	for idx := range pagedWorkers {
		saved, err := repo.Create(ctx, &worker.Worker{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			StateID:        data.State.ID,
			Status:         domaintypes.StatusActive,
			Type:           worker.WorkerTypeEmployee,
			DriverType:     worker.DriverTypeOTR,
			FirstName:      fmt.Sprintf("Pager%02d", idx),
			LastName:       fmt.Sprintf("Cursor%02d", pagedWorkers-idx),
			AddressLine1:   "1 Main St",
			City:           "Springfield",
			PostalCode:     "62701",
			Gender:         worker.GenderMale,
			Profile: &worker.WorkerProfile{
				DOB:           timeutils.NowUnix() - 86400*365*30,
				LicenseNumber: fmt.Sprintf("DL%06d", idx),
				LicenseExpiry: timeutils.NowUnix() + 86400*365,
				HireDate:      timeutils.NowUnix() - 86400*365,
			},
		})
		require.NoError(t, err)
		created[saved.ID.String()] = struct{}{}
	}

	tool := newListWorkersTool(repo, &fakePermissions{})
	params := &agentports.QueryToolParams{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Actor: &permissionports.RequestActor{
			UserID:         data.User.ID,
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
		},
	}

	for name, args := range map[string]map[string]any{
		"newest first":     {},
		"sorted by name":   {"sortBy": "lastName", "sortDirection": "asc"},
		"searched by text": {"query": "Pager"},
	} {
		t.Run(name, func(t *testing.T) {
			seen := make(map[string]struct{}, pagedWorkers)
			after := ""
			pages := 0
			for {
				call := map[string]any{"limit": float64(3)}
				for key, value := range args {
					call[key] = value
				}
				if after != "" {
					call["after"] = after
				}
				params.Params = call

				result, err := tool.Query(ctx, params)
				require.NoError(t, err)
				encoded, err := sonic.Marshal(result)
				require.NoError(t, err)
				var page pagedWorkerRows
				require.NoError(t, sonic.Unmarshal(encoded, &page))

				pages++
				for _, row := range page.Items {
					if _, ours := created[row.ID]; !ours {
						continue
					}
					_, repeated := seen[row.ID]
					require.False(t, repeated, "worker %s came back on page %d", row.ID, pages)
					seen[row.ID] = struct{}{}
				}
				assert.Equal(t, page.HasMore, page.NextCursor != "")
				if !page.HasMore {
					break
				}
				require.Less(t, pages, pagedWorkers*2, "paging never reached the last page")
				after = page.NextCursor
			}

			assert.GreaterOrEqual(t, pages, 2, "the roster spans more than one page")
			assert.Len(t, seen, pagedWorkers, "every worker is listed exactly once")
		})
	}
}
