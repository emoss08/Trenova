package common

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const MaxBulkRecords = 50

const maxBulkReportedFails = 5

const BulkOutcomeField = "outcome"

const bulkOutcomeLabel = "What happens"

type RecordLabels func(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) map[pulid.ID]string

type singleRecordTool interface {
	agentports.AgentTool
	agentports.ToolPreviewer
}

type RecordBatch struct {
	Param       string                  `json:"-"`
	SingleParam string                  `json:"-"`
	Resource    permission.Resource     `json:"-"`
	Kinds       []permission.RecordKind `json:"-"`
	Noun        string                  `json:"-"`
	Nouns       string                  `json:"-"`
	Verb        string                  `json:"-"`
	Past        string                  `json:"-"`
	Shared      []string                `json:"-"`
	Single      singleRecordTool        `json:"-"`
	// needsPerson is the refusal a run from anything but a person's approval
	// gets; nil when the single tool may run on its own, so the twin may too.
	NeedsPerson error        `json:"-"`
	Unchanged   string       `json:"-"`
	Labels      RecordLabels `json:"-"`
}

type batchItem struct {
	id      pulid.ID
	change  *agent.RecordChange
	refusal error
}

// property is the list of ids the batch takes, marked with the kinds of record
// they are, for a resource whose name does not stand for its records alone (a
// billing queue's items), or else with the resource when it has a prefix, so
// the runtime refuses an id of another kind before any of the batch is read.
func (b *RecordBatch) property(description string) map[string]any {
	property := map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyMinItems:    1,
		toolschema.KeyMaxItems:    MaxBulkRecords,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
		},
	}
	if len(b.Kinds) > 0 {
		agenttoolschema.OfKinds(property, b.Kinds...)
	} else if _, typed := b.Resource.IDPrefix(); typed {
		agenttoolschema.OfResource(property, b.Resource)
	}

	return toolschema.RecordSubset(b.Resource.String(), property)
}

func (b *RecordBatch) ids(
	tool agentports.AgentTool,
	params *agentports.ToolExecuteParams,
) ([]pulid.ID, error) {
	if err := GuardPreview(tool, params); err != nil {
		return nil, err
	}

	ids, err := RequirePulidSlice(params.Params, b.Param, MaxBulkRecords)
	if err != nil {
		return nil, err
	}

	return sliceutils.Dedupe(ids), nil
}

func (b *RecordBatch) singleParams(
	params *agentports.ToolExecuteParams,
	id pulid.ID,
) agentports.ToolExecuteParams {
	single := *params
	single.Params = make(map[string]any, len(b.Shared)+1)
	single.Params[b.SingleParam] = id.String()
	for _, key := range b.Shared {
		if value, ok := params.Params[key]; ok {
			single.Params[key] = value
		}
	}
	if params.IdempotencyKey != "" {
		single.IdempotencyKey = params.IdempotencyKey + ":" + id.String()
	}

	return single
}

func (b *RecordBatch) Preview(
	ctx context.Context,
	tool agentports.AgentTool,
	params *agentports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	ids, err := b.ids(tool, params)
	if err != nil {
		return nil, err
	}

	checked := ids[:min(len(ids), agent.MaxPreviewRecords)]
	items := make([]batchItem, 0, len(checked))
	for _, id := range checked {
		item, itemErr := b.previewOne(ctx, params, id)
		if itemErr != nil {
			return nil, itemErr
		}
		items = append(items, item)
	}
	slices.SortStableFunc(items, func(a, c batchItem) int {
		switch {
		case (a.refusal == nil) == (c.refusal == nil):
			return 0
		case a.refusal != nil:
			return -1
		default:
			return 1
		}
	})

	refused := make([]batchItem, 0, len(items))
	changes := make([]*agent.RecordChange, 0, len(items))
	for idx := range items {
		changes = append(changes, items[idx].change)
		if items[idx].refusal != nil {
			refused = append(refused, items[idx])
		}
	}

	preview := toolpreview.Build(b.summary(len(ids), len(checked), len(refused)), changes...)
	if len(ids) > len(checked) {
		preview.Partial = true
	}
	if len(refused) == len(items) {
		preview.AddWarning(toolpreview.WouldFail(b.refusalOf(refused)))
	}

	return preview, nil
}

func (b *RecordBatch) previewOne(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
	id pulid.ID,
) (batchItem, error) {
	single, err := b.Single.Preview(ctx, b.singleParams(params, id))
	if err != nil {
		if !IsRefusal(err) && !errortypes.IsNotFoundError(err) {
			return batchItem{}, err
		}

		return b.refusedItem(id, "", err), nil
	}

	primary := b.primaryChange(single, id)
	if refusal := toolpreview.Refused(single); refusal != nil {
		label := ""
		if primary != nil {
			label = primary.Label
		}

		return b.refusedItem(id, label, refusal), nil
	}
	if primary == nil {
		unchanged := b.Unchanged
		if unchanged == "" {
			unchanged = "would not change"
		}

		return b.refusedItem(id, "", fmt.Errorf("this %s %s", b.Noun, unchanged)), nil
	}
	primary.Fields = append(primary.Fields, outcomeField(single.Summary))

	return batchItem{id: id, change: primary}, nil
}

func (b *RecordBatch) primaryChange(preview *agent.ToolPreview, id pulid.ID) *agent.RecordChange {
	for idx := range preview.Changes {
		change := preview.Changes[idx]
		if change.Resource == b.Resource && change.EntityID == id {
			return &change
		}
	}

	return nil
}

func (b *RecordBatch) refusedItem(id pulid.ID, label string, refusal error) batchItem {
	return batchItem{
		id:      id,
		refusal: refusal,
		change: &agent.RecordChange{
			Resource:  b.Resource,
			EntityID:  id,
			Label:     label,
			Operation: agent.PreviewOperationUpdate,
			Fields: []agent.PreviewFieldChange{
				outcomeField("Refused: " + strings.TrimSpace(refusal.Error())),
			},
		},
	}
}

func outcomeField(text string) agent.PreviewFieldChange {
	return agent.PreviewFieldChange{
		Path:  BulkOutcomeField,
		Label: bulkOutcomeLabel,
		Type:  assistantartifact.DisplayText,
		After: strings.TrimSpace(text),
	}
}

func (b *RecordBatch) summary(total, checked, refused int) string {
	going := checked - refused
	counts := fmt.Sprintf("%d would be %s", going, b.Past)
	if refused > 0 {
		counts += fmt.Sprintf(", %d would be refused", refused)
	}
	if total > checked {
		return fmt.Sprintf(
			"Would %s %s. Of the first %d checked, %s; the rest are checked when it runs.",
			b.Verb, CountOf(total, b.Noun), checked, counts,
		)
	}

	return fmt.Sprintf("Would %s %s: %s.", b.Verb, CountOf(total, b.Noun), counts)
}

func (b *RecordBatch) refusalOf(refused []batchItem) error {
	reasons := make([]string, 0, min(len(refused), maxBulkReportedFails))
	for idx, item := range refused {
		if idx == maxBulkReportedFails {
			reasons = append(reasons, fmt.Sprintf("and %d more", len(refused)-idx))

			break
		}
		name := item.change.Label
		if name == "" {
			name = item.id.String()
		}
		reasons = append(reasons, name+": "+strings.TrimSpace(item.refusal.Error()))
	}

	return fmt.Errorf("none of the %s would be %s as they stand: %s",
		b.Nouns, b.Past, strings.Join(reasons, "; "))
}

func (b *RecordBatch) Validate(
	ctx context.Context,
	tool agentports.AgentTool,
	params *agentports.ToolExecuteParams,
) error {
	preview, err := b.Preview(ctx, tool, params)
	if err != nil {
		return err
	}

	return toolpreview.Refused(preview)
}

type batchFailure struct {
	id    pulid.ID
	cause error
}

func (b *RecordBatch) Execute(
	ctx context.Context,
	tool agentports.AgentTool,
	params *agentports.ToolExecuteParams,
	run func(ctx context.Context, params agentports.ToolExecuteParams) error,
) (*agent.ToolExecutionResult, error) {
	if err := GuardExecute(tool, *params); err != nil {
		return nil, err
	}
	if b.NeedsPerson != nil && !params.ApprovedFromProposal() {
		return nil, b.NeedsPerson
	}

	ids, err := b.ids(tool, params)
	if err != nil {
		return nil, err
	}

	failures := make([]batchFailure, 0, len(ids))
	for _, id := range ids {
		if runErr := run(ctx, b.singleParams(params, id)); runErr != nil {
			if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
				return nil, runErr
			}
			failures = append(failures, batchFailure{id: id, cause: runErr})
		}
	}

	outcome := b.outcome(ctx, params, len(ids), failures)
	if len(failures) == len(ids) {
		return nil, errors.New(outcome)
	}

	return &agent.ToolExecutionResult{
		Action: b.Past,
		Kind:   b.Nouns,
		Name:   outcome,
		Total:  len(ids),
		Failed: b.failedItems(ctx, params, failures),
	}, nil
}

// failedItems is each record the write did not change, named and with its
// reason, for the card that lists them.
func (b *RecordBatch) failedItems(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
	failures []batchFailure,
) []agent.ExecutionItemFail {
	if len(failures) == 0 {
		return nil
	}
	names := b.failedLabels(ctx, params, failures)
	out := make([]agent.ExecutionItemFail, 0, len(failures))
	for _, failure := range failures {
		out = append(out, agent.ExecutionItemFail{
			ID:     failure.id.String(),
			Label:  names[failure.id],
			Reason: strings.TrimSpace(failure.cause.Error()),
		})
	}

	return out
}

func (b *RecordBatch) outcome(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
	total int,
	failures []batchFailure,
) string {
	outcome := fmt.Sprintf("%d of %d %s", total-len(failures), total, b.Past)
	if len(failures) == 0 {
		return outcome
	}

	names := b.failedLabels(ctx, params, failures)
	reasons := make([]string, 0, min(len(failures), maxBulkReportedFails))
	for idx, failure := range failures {
		if idx == maxBulkReportedFails {
			reasons = append(reasons, fmt.Sprintf("and %d more", len(failures)-idx))

			break
		}
		name := names[failure.id]
		if name == "" {
			name = failure.id.String()
		}
		reasons = append(reasons, name+" ("+strings.TrimSpace(failure.cause.Error())+")")
	}

	return outcome + "; refused: " + strings.Join(reasons, ", ")
}

func (b *RecordBatch) failedLabels(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
	failures []batchFailure,
) map[pulid.ID]string {
	if b.Labels == nil {
		return map[pulid.ID]string{}
	}

	ids := make([]pulid.ID, 0, len(failures))
	for _, failure := range failures {
		ids = append(ids, failure.id)
	}
	labels := b.Labels(ctx, TenantFrom(*params), ids)
	if labels == nil {
		return map[pulid.ID]string{}
	}

	return labels
}
