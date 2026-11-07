package agentwaitports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RegisterWaitRequest parks the work of a conversation, or of a background
// run, until something happens.
type RegisterWaitRequest struct {
	Actor           *permissionports.RequestActor
	DefinitionID    pulid.ID
	ThreadID        pulid.ID
	RunID           pulid.ID
	Delegated       bool
	Kind            agentwait.Kind
	Condition       agentwait.Condition
	Description     string
	Then            string
	LifetimeSeconds int64
}

type CancelWaitRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	// ThreadID and UserID, when set, cancel the wait only as one of that
	// person's conversation's.
	ThreadID pulid.ID
	UserID   pulid.ID
	// DefinitionID, when set without a thread, cancels only a wait that
	// agent set: a background run never reaches another agent's waits.
	DefinitionID pulid.ID
	By           string
}

type WaitForDocumentReadRequest struct {
	Actor        *permissionports.RequestActor
	DefinitionID pulid.ID
	ThreadID     pulid.ID
	DocumentID   pulid.ID
	FileName     string
}

// DocumentReadWaiter parks a conversation on a file the person attached that
// is still being read, so the conversation picks up again on its own when the
// reading finishes rather than waiting for the person to ask again.
type DocumentReadWaiter interface {
	WaitForDocumentRead(
		ctx context.Context,
		req *WaitForDocumentReadRequest,
	) (*agentwait.Wait, error)
	// StopWaitingForDocumentRead cancels the conversation's wait on a file
	// the agent has since read in full, so it does not pick the work up
	// again for nothing.
	StopWaitingForDocumentRead(ctx context.Context, req *WaitForDocumentReadRequest) error
}

type AgentWaitService interface {
	Register(ctx context.Context, req *RegisterWaitRequest) (*agentwait.Wait, error)
	Cancel(ctx context.Context, req *CancelWaitRequest) (*agentwait.Wait, error)
}

// AgentWaitNotifier hands what happened in the world to the waits watching
// for it. It never fails the work that reported it.
type AgentWaitNotifier interface {
	NotifyEvent(ctx context.Context, event *agenteventports.AgentEvent)
	NotifyHOS(
		ctx context.Context,
		tenant pagination.TenantInfo,
		states []*telematics.WorkerHOSState,
	)
	// NotifyPositions checks the polled truck positions against the stops
	// that open waits on an arrival or a departure are watching.
	NotifyPositions(
		ctx context.Context,
		tenant pagination.TenantInfo,
		positions []*telematics.VehiclePosition,
	)
}

// FoundRecord is what a record finder knows of one record: whether it reads
// records of that kind at all, whether this one is the organization's, and
// what it is called.
type FoundRecord struct {
	Known  bool
	Exists bool
	Label  string
}

// RecordFinder reads whether a record a wait names is there and what it is
// called, and the names of the people who change watched records.
type RecordFinder interface {
	Find(
		ctx context.Context,
		tenant pagination.TenantInfo,
		resource permission.Resource,
		id pulid.ID,
	) (*FoundRecord, error)
	UserNames(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
	) (map[pulid.ID]string, error)
}

type PersonSpokeSinceRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Since      int64
}

// ConversationActivity reads what a conversation's owner did after a time:
// a change they made and then told the conversation about needs no second
// turn to pick it up.
type ConversationActivity interface {
	PersonSpokeSince(ctx context.Context, req *PersonSpokeSinceRequest) (bool, error)
}

// RecordChangeNotifier hands changes to watched records, read off the
// realtime change stream, to the waits watching them.
type RecordChangeNotifier interface {
	NotifyRecordChanges(ctx context.Context, changes []realtimeports.TenantRecordChange)
}

type WaitForReportRunRequest struct {
	Actor        *permissionports.RequestActor
	DefinitionID pulid.ID
	ThreadID     pulid.ID
	RunID        pulid.ID
	Delegated    bool
	ReportRunID  pulid.ID
	ReportName   string
	Then         string
}

// ReportRunWaiter parks the work that started a report run still going, so
// it picks up what the person asked of the report once the run finishes.
type ReportRunWaiter interface {
	WaitForReportRun(ctx context.Context, req *WaitForReportRunRequest) (*agentwait.Wait, error)
}
