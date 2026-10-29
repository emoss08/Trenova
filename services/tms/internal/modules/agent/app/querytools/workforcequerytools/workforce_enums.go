package workforcequerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
)

// The sources the read tools' enums are drawn from, where the list is not
// already a filter's own values.
var (
	ptoStatuses = agenttoolschema.Source("worker.ptoStatus", []worker.PTOStatus{
		worker.PTOStatusRequested,
		worker.PTOStatusApproved,
		worker.PTOStatusRejected,
		worker.PTOStatusCancelled,
	})
)
