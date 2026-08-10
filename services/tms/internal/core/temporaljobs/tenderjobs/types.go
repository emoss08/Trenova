package tenderjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	CarrierTenderWorkflowName = "CarrierTenderWorkflow"
	TenderSignalName          = "tender_signal"

	WorkflowIDPrefix = "carrier-tender-"
)

// WorkflowID derives the deterministic workflow identifier for one tender.
func WorkflowID(tenderID pulid.ID) string {
	return WorkflowIDPrefix + tenderID.String()
}

type WorkflowInput struct {
	TenderID       pulid.ID              `json:"tenderId"`
	OrganizationID pulid.ID              `json:"organizationId"`
	BusinessUnitID pulid.ID              `json:"businessUnitId"`
	Mode           tender.Mode           `json:"mode"`
	StartAtOfferID pulid.ID              `json:"startAtOfferId"`
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

const (
	SignalKindResponse   = SignalKind("Response")
	SignalKindCancel     = SignalKind("Cancel")
	SignalKindSweepNudge = SignalKind("SweepNudge")
)

// Signal is the single message shape every channel delivers to the workflow.
// The workflow is the sole owner of tender state transitions, so a response
// never mutates rows before the workflow drains it.
type Signal struct {
	Kind SignalKind `json:"kind"`

	OfferID       pulid.ID              `json:"offerId"`
	Action        tender.ResponseAction `json:"action"`
	Source        tender.ResponseSource `json:"source"`
	DeclineReason string                `json:"declineReason"`

	Reason      string   `json:"reason"`
	ActorUserID pulid.ID `json:"actorUserId"`
}
