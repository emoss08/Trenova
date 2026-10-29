package edidecision

import (
	"github.com/emoss08/trenova/shared/pulid"
)

func buildLoadTenderApprovalWorkflowID(transferID pulid.ID) string {
	return "edi-load-tender-approve-" + transferID.String()
}
