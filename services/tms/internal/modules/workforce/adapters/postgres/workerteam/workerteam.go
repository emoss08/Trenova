package workerteam

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/pulid"
)

func Where(workerID keel.Col, managerIDs []pulid.ID) keel.Pred {
	if len(managerIDs) == 0 {
		return nil
	}

	return workerID.InQuery(Members(managerIDs))
}

func Members(managerIDs []pulid.ID) *keel.Query[*worker.Worker] {
	w := workerdb.Workers.As("team_workers")

	return w.Pick(w.ID).Where(keel.Or(
		w.ManagerID.In(managerIDs...),
		w.FleetCodeID.InQuery(managedFleets(managerIDs)),
	))
}

func managedFleets(managerIDs []pulid.ID) *keel.Query[*fleetcode.FleetCode] {
	fc := fleetcodedb.FleetCodes.As("team_fleets")

	return fc.Pick(fc.ID).Where(fc.ManagerID.In(managerIDs...))
}
