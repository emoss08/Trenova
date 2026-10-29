package recordbatch

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
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
	agenttoolports.AgentTool
	agenttoolports.ToolPreviewer
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
	change  *agentpreview.RecordChange
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
	tool agenttoolports.AgentTool,
	params *agenttoolports.ToolExecuteParams,
) ([]pulid.ID, error) {
	if err := toolguard.GuardPreview(tool, params); err != nil {
		return nil, err
	}

	ids, err := toolparams.RequirePulidSlice(params.Params, b.Param, MaxBulkRecords)
	if err != nil {
		return nil, err
	}

	return sliceutils.Dedupe(ids), nil
}

func (b *RecordBatch) singleArguments(params map[string]any, id pulid.ID) map[string]any {
	arguments := make(map[string]any, len(b.Shared)+1)
	arguments[b.SingleParam] = id.String()
	for _, key := range b.Shared {
		if value, ok := params[key]; ok {
			arguments[key] = value
		}
	}

	return arguments
}

func (b *RecordBatch) singleParams(
	params *agenttoolports.ToolExecuteParams,
	id pulid.ID,
) agenttoolports.ToolExecuteParams {
	single := *params
	single.Params = b.singleArguments(params.Params, id)
	if params.IdempotencyKey != "" {
		single.IdempotencyKey = params.IdempotencyKey + ":" + id.String()
	}

	return single
}

// Split is the call naming each record alone, in the order named and each
// once, with the parameters every record shares.
func (b *RecordBatch) Split(params map[string]any) ([]map[string]any, error) {
	ids, err := toolparams.RequirePulidSlice(params, b.Param, MaxBulkRecords)
	if err != nil {
		return nil, err
	}
	ids = sliceutils.Dedupe(ids)

	calls := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		call := make(map[string]any, len(b.Shared)+1)
		call[b.Param] = []any{id.String()}
		for _, key := range b.Shared {
			if value, ok := params[key]; ok {
				call[key] = value
			}
		}
		calls = append(calls, call)
	}

	return calls, nil
}

func (b *RecordBatch) Preview(
	ctx context.Context,
	tool agenttoolports.AgentTool,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	ids, err := b.ids(tool, params)
	if err != nil {
		return nil, err
	}
	if len(ids) == 1 {
		return b.Single.Preview(ctx, b.singleParams(params, ids[0]))
	}

	checked := ids[:min(len(ids), agentpreview.MaxPreviewRecords)]
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
	changes := make([]*agentpreview.RecordChange, 0, len(items))
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
	params *agenttoolports.ToolExecuteParams,
	id pulid.ID,
) (batchItem, error) {
	single, err := b.Single.Preview(ctx, b.singleParams(params, id))
	if err != nil {
		if !changepreview.IsRefusal(err) && !errortypes.IsNotFoundError(err) {
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

func (b *RecordBatch) primaryChange(
	preview *agentpreview.ToolPreview,
	id pulid.ID,
) *agentpreview.RecordChange {
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
		change: &agentpreview.RecordChange{
			Resource:  b.Resource,
			EntityID:  id,
			Label:     label,
			Operation: agentpreview.PreviewOperationUpdate,
			Fields: []agentpreview.PreviewFieldChange{
				outcomeField("Refused: " + strings.TrimSpace(refusal.Error())),
			},
		},
	}
}

func outcomeField(text string) agentpreview.PreviewFieldChange {
	return agentpreview.PreviewFieldChange{
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
			b.Verb, changepreview.CountOf(total, b.Noun), checked, counts,
		)
	}

	return fmt.Sprintf("Would %s %s: %s.", b.Verb, changepreview.CountOf(total, b.Noun), counts)
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
	tool agenttoolports.AgentTool,
	params *agenttoolports.ToolExecuteParams,
) error {
	if validator, validates := b.Single.(agenttoolports.ToolValidator); validates {
		ids, err := b.ids(tool, params)
		if err != nil {
			return err
		}
		if len(ids) == 1 {
			return validator.Validate(ctx, b.singleParams(params, ids[0]))
		}
	}

	preview, err := b.Preview(ctx, tool, params)
	if err != nil {
		return err
	}

	return toolpreview.Refused(preview)
}

// Target is the record a call naming one record acts on, as the single tool
// names it, so a proposal for one record is pinned to its version like the
// single tool's was. A call naming several has no one target.
func (b *RecordBatch) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	targeted, ok := b.Single.(agenttoolports.TargetedTool)
	if !ok {
		return agenttoolports.ToolTarget{}, false
	}
	ids, err := toolparams.RequirePulidSlice(params, b.Param, MaxBulkRecords)
	if err != nil {
		return agenttoolports.ToolTarget{}, false
	}
	ids = sliceutils.Dedupe(ids)
	if len(ids) != 1 {
		return agenttoolports.ToolTarget{}, false
	}

	return targeted.Target(b.singleArguments(params, ids[0]))
}

type batchFailure struct {
	id    pulid.ID
	cause error
}

func (b *RecordBatch) Execute(
	ctx context.Context,
	tool agenttoolports.AgentTool,
	params *agenttoolports.ToolExecuteParams,
) (*agentrecord.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(tool, *params); err != nil {
		return nil, err
	}
	if b.NeedsPerson != nil && !params.ApprovedFromProposal() {
		return nil, b.NeedsPerson
	}

	ids, err := b.ids(tool, params)
	if err != nil {
		return nil, err
	}
	if len(ids) == 1 {
		return b.executeOne(ctx, params, ids[0])
	}

	failures := make([]batchFailure, 0, len(ids))
	for _, id := range ids {
		if runErr := b.Single.Execute(ctx, b.singleParams(params, id)); runErr != nil {
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

	return &agentrecord.ToolExecutionResult{
		Action: b.Past,
		Kind:   b.Nouns,
		Name:   outcome,
		Total:  len(ids),
		Failed: b.failedItems(ctx, params, failures),
	}, nil
}

// executeOne runs a call naming one record as the single tool's own call:
// its result when it reports one, and its own refusal when it fails, rather
// than a count of one.
func (b *RecordBatch) executeOne(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	id pulid.ID,
) (*agentrecord.ToolExecutionResult, error) {
	result, err := agenttoolports.ExecuteTool(ctx, b.Single, b.singleParams(params, id))
	if err != nil || result != nil {
		return result, err
	}

	return &agentrecord.ToolExecutionResult{
		Action: b.Past,
		Kind:   b.Nouns,
		Name:   b.outcome(ctx, params, 1, nil),
		Total:  1,
	}, nil
}

// failedItems is each record the write did not change, named and with its
// reason, for the card that lists them.
func (b *RecordBatch) failedItems(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	failures []batchFailure,
) []agentrecord.ExecutionItemFail {
	if len(failures) == 0 {
		return nil
	}
	names := b.failedLabels(ctx, params, failures)
	out := make([]agentrecord.ExecutionItemFail, 0, len(failures))
	for _, failure := range failures {
		out = append(out, agentrecord.ExecutionItemFail{
			ID:     failure.id.String(),
			Label:  names[failure.id],
			Reason: strings.TrimSpace(failure.cause.Error()),
		})
	}

	return out
}

func (b *RecordBatch) outcome(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
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
	params *agenttoolports.ToolExecuteParams,
	failures []batchFailure,
) map[pulid.ID]string {
	if b.Labels == nil {
		return map[pulid.ID]string{}
	}

	ids := make([]pulid.ID, 0, len(failures))
	for _, failure := range failures {
		ids = append(ids, failure.id)
	}
	labels := b.Labels(ctx, toolparams.TenantFrom(*params), ids)
	if labels == nil {
		return map[pulid.ID]string{}
	}

	return labels
}
