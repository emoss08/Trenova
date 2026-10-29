package billingqueuepreview

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	ToolGetBillingQueueItem = "billing_get_queue_item"
)

const FieldReviewCompletedAt = "reviewCompletedAt"

const FieldReviewStartedAt = "reviewStartedAt"

// billingQueueRecordEntity is the record-link registry's name for a
// billing queue item, which a transfer makes one of per payer.
const BillingQueueRecordEntity = "billing_queue_item"

var DecisionVolatile = []string{FieldReviewCompletedAt, "canceledAt", FieldReviewStartedAt}

var DecisionRefs = map[string]permission.Resource{
	changepreview.FieldCanceledByID: permission.ResourceUser,
	"assignedBillerId":              permission.ResourceUser,
}
