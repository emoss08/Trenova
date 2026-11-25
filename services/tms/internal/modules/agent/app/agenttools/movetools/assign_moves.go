package movetools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/recordbatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramShipmentMoveIDs   = "shipmentMoveIds"
	paramAssignments       = "assignments"
	paramPrimaryWorkerID   = "primaryWorkerId"
	paramSecondaryWorkerID = "secondaryWorkerId"
	paramTrailerID         = "trailerId"
)

const assignMovesDescription = "Assign a driver and equipment to one shipment move or to many " +
	"in one call the person decides once, each move with its own driver. Use it once each driver " +
	"is chosen: dispatch_plan hands back an assignments list ready to send for a set of " +
	"shipments or moves, and dispatch_rank_candidates gives the choices for one. Send " +
	"every move you want covered in one call rather than a call per move; the person " +
	"approving may drop any of them. Each move must have no driver yet; one the service " +
	"would refuse is reported and the rest still go. The same driver or tractor on two moves " +
	"of one call is flagged for the person to check. Once it runs, its result says who is on " +
	"each move with what equipment, so no read is needed to confirm it."

const assignmentsDescription = "One entry per move to cover: the move, the driver, the " +
	"tractor and, when the move needs one, the trailer, with why this driver. Take each " +
	"from dispatch_plan's assignments or from dispatch_rank_candidates; never send a " +
	"shipment id for a move id."

const assignMoveIDsDescription = "The moves this call covers, one for each entry in " +
	"assignments, which come from dispatch_plan or dispatch_rank_candidates. Leave it out: " +
	"the runtime fills it from assignments, and the person approving may drop moves from it."

// artifactShipment is the record-link key of what the batch reports: a move
// it could not assign is named as its shipment.
const artifactShipment = "shipment"

const (
	nounMove   = "move"
	nounMoves  = "moves"
	verbAssign = "assign a driver to"
	pastAssign = "assigned"
)

// AssignMovesTool assigns drivers to one move or many: one proposal, one card,
// one decision, each move assigned as AssignMoveTool would assign it alone.
type AssignMovesTool struct {
	single *AssignMoveTool
	batch  *recordbatch.RecordBatch
}

func NewAssignMovesTool(
	assignments assignmentports.AssignmentService,
	console dispatchconsoleports.DispatchConsoleService,
) agenttoolports.AgentTool {
	single, _ := NewAssignMoveTool(assignments, console).(*AssignMoveTool)

	return &AssignMovesTool{
		single: single,
		batch: &recordbatch.RecordBatch{
			Param:       paramShipmentMoveIDs,
			SingleParam: toolparams.ParamShipmentMoveID,
			Rows:        paramAssignments,
			Resource:    permission.ResourceShipmentMove,
			Noun:        nounMove,
			Nouns:       nounMoves,
			Verb:        verbAssign,
			Past:        pastAssign,
			Single:      single,
			Unchanged:   "already has this driver",
			Owners:      moveShipments(console),
		},
	}
}

// moveShipments reports each move a batch could not assign as its shipment,
// under its PRO number, so a failure is said as the dispatcher knows the load
// and links to its shipment. It reads only the moves that failed, never the
// whole batch.
func moveShipments(console dispatchconsoleports.DispatchConsoleService) recordbatch.RecordOwners {
	if console == nil {
		return nil
	}

	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
	) map[pulid.ID]recordbatch.RecordOwner {
		owners := make(map[pulid.ID]recordbatch.RecordOwner, len(ids))
		for _, id := range ids {
			move, err := console.GetMove(ctx, tenant, id)
			if err != nil || move == nil || move.BoardMove == nil {
				continue
			}
			owners[id] = recordbatch.RecordOwner{ID: move.ShipmentID, Label: move.ProNumber}
		}

		return owners
	}
}

func (t *AssignMovesTool) Name() string { return "dispatch_assign_move" }

// Prerequisites are the single tool's reads and the plan, which covers many
// moves in one call and hands back the rows this tool takes.
func (t *AssignMovesTool) Prerequisites() []string {
	return append([]string{"dispatch_plan"}, t.single.Prerequisites()...)
}

func (t *AssignMovesTool) Recipe() []string {
	return []string{"shipment_list", "dispatch_plan", "dispatch_assign_move"}
}

func (t *AssignMovesTool) SearchTerms() []string {
	return []string{recordbatch.SearchTermBulk}
}

func (t *AssignMovesTool) Description() string { return assignMovesDescription }

func (t *AssignMovesTool) ParamSchema() map[string]any {
	return recordbatch.BulkRowsSchema(
		t.batch,
		assignMoveIDsDescription,
		assignmentsDescription,
		t.single.ParamSchema(),
		nil,
	)
}

func (t *AssignMovesTool) Policy() agenttoolports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = artifactShipment
	policy.Condition = &agenttoolports.TierCondition{
		Description: "Only a call covering one move may run without a decision; several moves " +
			"always wait for a person, who may drop any of them.",
		Limit: t.tierLimit,
	}

	return policy
}

// tierLimit holds a call covering several moves for a person. A call whose
// moves cannot be read is not several: validation refuses it before it is
// filed and execution refuses it again, so nothing of it can run on its own.
func (t *AssignMovesTool) tierLimit(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) agent.AutonomyTier {
	calls, err := t.batch.Split(params.Params)
	if err == nil && len(calls) > 1 {
		return agent.TierActWithApproval
	}

	return agent.TierAutoExecute
}

func (t *AssignMovesTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return t.batch.Target(params)
}

func (t *AssignMovesTool) SplitRecords(params map[string]any) ([]map[string]any, error) {
	return t.batch.Split(params)
}

// ResolveSelection writes the moves the call covers into shipmentMoveIds, so
// the person approving sees each as a move they may drop.
func (t *AssignMovesTool) ResolveSelection(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) (map[string]any, error) {
	return t.batch.PinSelection(params.Params)
}

func (t *AssignMovesTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	return t.batch.Validate(ctx, t, params)
}

func (t *AssignMovesTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	preview, err := t.batch.Preview(ctx, t, params)
	if err != nil || preview == nil {
		return preview, err
	}
	if warning, loads := t.doubleBooked(params.Params, moveLoads(preview)); warning != "" {
		toolpreview.Warn(preview, agentpreview.PreviewWarningDoubleBooked, warning, loads...)
	}

	return preview, nil
}

func (t *AssignMovesTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

// ExecuteWithResult assigns each kept move and, for several, reads them all
// back in one read, so the agent's next turn says who is on each load
// rather than a count it would have to check shipment by shipment.
func (t *AssignMovesTool) ExecuteWithResult(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentrecord.ToolExecutionResult, error) {
	result, err := t.batch.Execute(ctx, t, params)
	if err != nil || result == nil || result.Total <= 1 {
		return result, err
	}
	moveIDs, idsErr := t.batch.RecordIDs(params.Params)
	if idsErr == nil {
		confirmAssignments(ctx, t.single.console, toolparams.TenantFrom(params), result, moveIDs)
	}

	return result, nil
}

// moveLoads is each previewed move's load as the preview names it ("Move on
// S26110993231548" is the load S26110993231548), keyed by the move.
func moveLoads(preview *agentpreview.ToolPreview) map[string]string {
	loads := make(map[string]string, len(preview.Changes))
	for idx := range preview.Changes {
		change := &preview.Changes[idx]
		if change.Resource != permission.ResourceShipmentMove || change.EntityID.IsNil() {
			continue
		}
		label := strings.TrimSpace(strings.TrimPrefix(change.Label, movePreviewLabelPrefix))
		if label != "" {
			loads[change.EntityID.String()] = label
		}
	}

	return loads
}

// movePreviewLabelPrefix is how shipmentpreview.CoverageChanges labels the
// move it covers: "Move on " and the shipment's PRO number.
const movePreviewLabelPrefix = "Move on "

// doubleBooked says which loads of the call share a driver or a tractor. The
// assignment service checks each move alone, so two moves of one call can each
// pass with the same driver; the person approving decides whether the loads
// follow one another or one of them needs someone else. It names loads, never
// a driver's or tractor's id.
func (t *AssignMovesTool) doubleBooked(
	params map[string]any,
	loads map[string]string,
) (string, []string) {
	calls, err := t.batch.Split(params)
	if err != nil || len(calls) < 2 {
		return "", nil
	}

	type use struct {
		kind  string
		moves []string
	}
	uses := make(map[string]*use)
	order := make([]string, 0)
	note := func(kind, id, move string) {
		if id == "" {
			return
		}
		key := kind + ":" + id
		if uses[key] == nil {
			uses[key] = &use{kind: kind}
			order = append(order, key)
		}
		uses[key].moves = append(uses[key].moves, move)
	}
	for _, call := range calls {
		rows, _ := call[paramAssignments].([]any)
		if len(rows) != 1 {
			continue
		}
		row, _ := rows[0].(map[string]any)
		move := toolparams.OptionalString(row, toolparams.ParamShipmentMoveID)
		note(kindDriver, toolparams.OptionalString(row, paramPrimaryWorkerID), move)
		note(kindDriver, toolparams.OptionalString(row, paramSecondaryWorkerID), move)
		note(kindTractor, toolparams.OptionalString(row, toolparams.ParamTractorID), move)
	}

	parts := make([]string, 0)
	named := make([]string, 0)
	for _, key := range order {
		entry := uses[key]
		if len(entry.moves) < 2 {
			continue
		}
		names := make([]string, 0, len(entry.moves))
		for _, move := range entry.moves {
			name := loads[move]
			if name == "" {
				name = "a load not previewed"
			} else if !slices.Contains(named, name) {
				named = append(named, name)
			}
			names = append(names, name)
		}
		parts = append(parts, fmt.Sprintf("one %s on %s", entry.kind, joinAnd(names)))
	}
	if len(parts) == 0 {
		return "", nil
	}

	return "The same driver or tractor is on several loads: " + strings.Join(parts, "; ") +
		". Each load was checked alone, so keep them together only when one follows the " +
		"other; otherwise drop one and cover it with someone else.", named
}

const (
	kindDriver  = "driver"
	kindTractor = "tractor"
)

func joinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}
