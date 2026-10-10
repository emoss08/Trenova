package bulkeditcontract

import (
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const BulkEditWorkflowName = "BulkEditWorkflow"

const WorkflowIDPrefix = "bulk-edit-"

type EditPayload struct {
	temporaltype.BasePayload

	EditID pulid.ID `json:"editId"`
}
