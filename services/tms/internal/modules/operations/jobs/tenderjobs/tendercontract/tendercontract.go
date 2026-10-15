package tendercontract

import (
	"strconv"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const CarrierTenderWorkflowName = "CarrierTenderWorkflow"

const TenderSignalName = "tender_signal"

const WorkflowIDPrefix = "carrier-tender-"

// WorkflowID derives the deterministic workflow identifier for one round of a
// tender. A carrier walking an acceptance back starts the next round under its
// own identifier, so it never waits on the round that took the acceptance to
// finish closing out.
func WorkflowID(tenderID pulid.ID, round int16) string {
	if round <= 0 {
		return WorkflowIDPrefix + tenderID.String()
	}
	return WorkflowIDPrefix + tenderID.String() + "-r" + strconv.Itoa(int(round))
}

type WorkflowInput struct {
	TenderID       pulid.ID    `json:"tenderId"`
	OrganizationID pulid.ID    `json:"organizationId"`
	BusinessUnitID pulid.ID    `json:"businessUnitId"`
	Mode           tender.Mode `json:"mode"`
}

func (in WorkflowInput) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: in.OrganizationID,
		BuID:  in.BusinessUnitID,
	}
}

// SignalKind discriminates the messages delivered on the tender signal
// channel.
type SignalKind string

const SignalKindResponse = SignalKind("Response")

const SignalKindCancel = SignalKind("Cancel")

// Signal is the single message shape every channel delivers to the workflow.
// The workflow is the sole owner of tender state transitions, so a response
// never mutates rows before the workflow drains it.
type Signal struct {
	Kind SignalKind `json:"kind"`

	OfferID       pulid.ID              `json:"offerId"`
	Action        tender.ResponseAction `json:"action"`
	Source        tender.ResponseSource `json:"source"`
	DeclineReason string                `json:"declineReason"`
	CounterRate   decimal.Decimal       `json:"counterRate"`
	CounterNote   string                `json:"counterNote"`

	Reason      string   `json:"reason"`
	ActorUserID pulid.ID `json:"actorUserId"`
}
