package workerteamtest

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type Team struct {
	Manager  *tenant.User
	Direct   *worker.Worker
	ViaFleet *worker.Worker
	Outsider *worker.Worker
}

func (t *Team) ManagerIDs() []pulid.ID { return []pulid.ID{t.Manager.ID} }

func Seed(t *testing.T, ctx context.Context, db *keel.DB, data *seedtest.TestData) *Team {
	t.Helper()

	manager := seedtest.NewUser(data.Organization.ID, data.BusinessUnit.ID).
		WithUsername("team-manager").
		WithEmail("team-manager@example.com").
		Build(t, ctx, db)
	tenantCtx := data.Ctx(ctx)

	fleet := &fleetcode.FleetCode{
		ID:             pulid.MustNew("fc_"),
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		ManagerID:      manager.ID,
		Status:         domaintypes.StatusActive,
		Code:           "TEAM",
	}
	require.NoError(t, fleetcodedb.FleetCodes.Create(tenantCtx, db, fleet))

	newWorker := func(first string, managerID, fleetID pulid.ID) *worker.Worker {
		wrk := &worker.Worker{
			ID:             pulid.MustNew("wrk_"),
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			StateID:        data.State.ID,
			Status:         domaintypes.StatusActive,
			Type:           worker.WorkerTypeEmployee,
			DriverType:     worker.DriverTypeOTR,
			FirstName:      first,
			LastName:       "Driver",
			AddressLine1:   "1 Main St",
			City:           "Springfield",
			PostalCode:     "62701",
			Gender:         worker.GenderFemale,
			ManagerID:      managerID,
			FleetCodeID:    fleetID,
		}
		require.NoError(t, workerdb.Workers.Create(tenantCtx, db, wrk))
		return wrk
	}

	return &Team{
		Manager:  manager,
		Direct:   newWorker("Direct", manager.ID, pulid.Nil),
		ViaFleet: newWorker("Fleet", pulid.Nil, fleet.ID),
		Outsider: newWorker("Outsider", data.User.ID, pulid.Nil),
	}
}
