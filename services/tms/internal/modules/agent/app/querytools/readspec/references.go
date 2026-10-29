package readspec

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

var (
	CustomerReference = Reference{
		Resource: permission.ResourceCustomer,
		What:     "The customer",
		Source:   querytool.ToolCustomerList,
	}
)

// WorkerReference is a worker id parameter: a worker, from the worker list.
var WorkerReference = Reference{
	Resource: permission.ResourceWorker,
	What:     "The worker",
	Source:   querytool.ToolWorkerList,
}

// TheWorker is the worker a per-worker list is keyed by.
var TheWorker = map[string]Reference{querytool.ParamWorkerID: WorkerReference}
