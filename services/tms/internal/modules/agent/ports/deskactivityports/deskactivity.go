package deskactivityports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/deskattentionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Outcome string

const (
	OutcomeOnItsOwn  = Outcome("OnItsOwn")
	OutcomeApproved  = Outcome("Approved")
	OutcomePreviewed = Outcome("Previewed")
	OutcomeFailed    = Outcome("Failed")
)

type ActivityRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	First      int
}

type Entry struct {
	ID                pulid.ID
	Outcome           Outcome
	At                int64
	AgentDefinitionID pulid.ID
	RunID             pulid.ID
	PlanID            pulid.ID
	Title             string
	Detail            string
	Count             int
	Records           []*deskattentionports.Record
	RecordsMore       int
	DecidedByUserID   pulid.ID
	ProposalIDs       []pulid.ID
}

type Activity struct {
	Entries []*Entry
	HasMore bool
}

type EntryRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type Change struct {
	ProposalID pulid.ID
	Record     *deskattentionports.Record
	What       string
	Why        string
	ProposedAt int64
	At         int64
	Outcome    Outcome
	Error      string
}

type EntryDetail struct {
	Entry     *Entry
	ToolName  string
	Changes   []*Change
	RanBefore int
	RanSince  int64
}

type DeskActivityService interface {
	Activity(
		ctx context.Context,
		req *ActivityRequest,
		actor *permissionports.RequestActor,
	) (*Activity, error)
	Entry(
		ctx context.Context,
		req *EntryRequest,
		actor *permissionports.RequestActor,
	) (*EntryDetail, error)
}
