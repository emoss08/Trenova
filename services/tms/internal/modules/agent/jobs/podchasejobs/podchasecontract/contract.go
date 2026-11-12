package podchasecontract

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentrunservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// DrainKind queues a shipment whose proof of delivery is to be asked
	// for; the item's detail is the agent the request is filed for.
	DrainKind         = "pod-chase"
	DrainWorkflowName = "PODChaseDrainWorkflow"
	DrainSignalName   = "pod-chase-drain"
	drainSummary      = "Ask for the proofs of delivery the tenant's delivered shipments are missing"
)

type RoutineParams struct {
	fx.In

	Queue   workqueueports.Queue
	Signals kernelports.WorkflowSignalStarter
	Logger  *zap.Logger
}

// Routine takes the built-in billing agent's missing-proof-of-delivery
// events in code: the request is the same for every shipment, so a page of
// them is asked for together and the agent's model is kept for a shipment
// nobody can be asked about.
type Routine struct {
	drain *temporaljobs.DrainRequester
}

var _ agenteventports.EventRoutine = (*Routine)(nil)

func NewRoutine(p RoutineParams) *Routine {
	return &Routine{drain: &temporaljobs.DrainRequester{
		Queue:     p.Queue,
		Signals:   p.Signals,
		Kind:      DrainKind,
		Workflow:  DrainWorkflowName,
		TaskQueue: temporaltype.TaskQueueAgentBackground.String(),
		Signal:    DrainSignalName,
		Summary:   drainSummary,
		Logger:    p.Logger.Named("pod-chase-routine"),
	}}
}

// Handles is the built-in billing exception agent's missing proof of
// delivery. An agent an organization wrote keeps its own model, whatever it
// was told to do with the event.
func (r *Routine) Handles(
	definition *agentdefinition.Definition,
	kind agenttrigger.EventKind,
) bool {
	return kind == agenttrigger.EventShipmentPODMissing && definition != nil &&
		definition.SystemKey == agentrunservice.SystemKeyBillingException
}

func (r *Routine) Take(
	ctx context.Context,
	definition *agentdefinition.Definition,
	event *agenteventports.AgentEvent,
) error {
	return r.drain.RequestWithDetail(ctx, event.TenantInfo, definition.ID.String(), event.SubjectID)
}

// WakeDue wakes every tenant with shipments still queued: the backstop for a
// wake that was lost.
func (r *Routine) WakeDue(ctx context.Context, limit int) (int, error) {
	return r.drain.WakeDue(ctx, limit)
}
