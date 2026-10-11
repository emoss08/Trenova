package common

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const ParamBillingQueueItemID = "billingQueueItemId"

const ParamReviewNotes = "reviewNotes"

const ParamCancelReason = "cancelReason"

const FieldCanceledByID = "canceledById"

const MoneyLineFreight = "Freight"

const MoneyLineAccessorials = "Accessorials"

const ResultCreated = "created"

const MaxDecisionNoteChars = 2000

// ErrDecisionNeedsAPerson is an approval or a cancellation that did not come
// from a proposal a person approved. The tool's tier ceiling already keeps the
// runtime from running it on its own, and the billing queue refuses an agent
// either decision; this is the same rule where the tool runs.
var ErrDecisionNeedsAPerson = errors.New(
	"a billing queue item is approved or canceled only once a person approves the proposal",
)

type BillingQueueDecider interface {
	GetByID(
		ctx context.Context,
		req *billingqueueports.GetBillingQueueItemByIDRequest,
	) (*billingqueue.BillingQueueItem, error)
	UpdateStatus(
		ctx context.Context,
		req *billingqueueports.UpdateBillingQueueStatusRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	AssignBiller(
		ctx context.Context,
		req *billingqueueports.AssignBillerRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	CheckBiller(ctx context.Context, tenantInfo pagination.TenantInfo, billerID pulid.ID) error
}
