package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	maxBulkRecords       = 50
	maxBulkReportedFails = 5
	bulkOutcomeField     = "outcome"
	bulkOutcomeLabel     = "What happens"
)

type recordLabels func(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) map[pulid.ID]string

type singleRecordTool interface {
	serviceports.AgentTool
	serviceports.ToolPreviewer
}

type recordBatch struct {
	param       string
	singleParam string
	resource    permission.Resource
	noun        string
	nouns       string
	verb        string
	past        string
	shared      []string
	single      singleRecordTool
	needsPerson error
	unchanged   string
	labels      recordLabels
}

type batchItem struct {
	id      pulid.ID
	change  *agent.RecordChange
	refusal error
}

func (b *recordBatch) property(description string) map[string]any {
	return toolschema.RecordSubset(b.resource.String(), map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyMinItems:    1,
		toolschema.KeyMaxItems:    maxBulkRecords,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
		},
	})
}

func (b *recordBatch) ids(
	tool serviceports.AgentTool,
	params *serviceports.ToolExecuteParams,
) ([]pulid.ID, error) {
	if err := guardPreview(tool, params); err != nil {
		return nil, err
	}

	ids, err := requirePulidSlice(params.Params, b.param, maxBulkRecords)
	if err != nil {
		return nil, err
	}

	return sliceutils.Dedupe(ids), nil
}

func (b *recordBatch) singleParams(
	params *serviceports.ToolExecuteParams,
	id pulid.ID,
) serviceports.ToolExecuteParams {
	single := *params
	single.Params = make(map[string]any, len(b.shared)+1)
	single.Params[b.singleParam] = id.String()
	for _, key := range b.shared {
		if value, ok := params.Params[key]; ok {
			single.Params[key] = value
		}
	}
	if params.IdempotencyKey != "" {
		single.IdempotencyKey = params.IdempotencyKey + ":" + id.String()
	}

	return single
}

func (b *recordBatch) Preview(
	ctx context.Context,
	tool serviceports.AgentTool,
	params *serviceports.ToolExecuteParams,
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

func (b *recordBatch) previewOne(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
	id pulid.ID,
) (batchItem, error) {
	single, err := b.single.Preview(ctx, b.singleParams(params, id))
	if err != nil {
		if !isRefusal(err) && !errortypes.IsNotFoundError(err) {
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
		unchanged := b.unchanged
		if unchanged == "" {
			unchanged = "would not change"
		}

		return b.refusedItem(id, "", fmt.Errorf("this %s %s", b.noun, unchanged)), nil
	}
	primary.Fields = append(primary.Fields, outcomeField(single.Summary))

	return batchItem{id: id, change: primary}, nil
}

func (b *recordBatch) primaryChange(preview *agent.ToolPreview, id pulid.ID) *agent.RecordChange {
	for idx := range preview.Changes {
		change := preview.Changes[idx]
		if change.Resource == b.resource && change.EntityID == id {
			return &change
		}
	}

	return nil
}

func (b *recordBatch) refusedItem(id pulid.ID, label string, refusal error) batchItem {
	return batchItem{
		id:      id,
		refusal: refusal,
		change: &agent.RecordChange{
			Resource:  b.resource,
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
		Path:  bulkOutcomeField,
		Label: bulkOutcomeLabel,
		Type:  assistantartifact.DisplayText,
		After: strings.TrimSpace(text),
	}
}

func (b *recordBatch) summary(total, checked, refused int) string {
	going := checked - refused
	counts := fmt.Sprintf("%d would be %s", going, b.past)
	if refused > 0 {
		counts += fmt.Sprintf(", %d would be refused", refused)
	}
	if total > checked {
		return fmt.Sprintf(
			"Would %s %s. Of the first %d checked, %s; the rest are checked when it runs.",
			b.verb, countOf(total, b.noun), checked, counts,
		)
	}

	return fmt.Sprintf("Would %s %s: %s.", b.verb, countOf(total, b.noun), counts)
}

func (b *recordBatch) refusalOf(refused []batchItem) error {
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
		b.nouns, b.past, strings.Join(reasons, "; "))
}

func (b *recordBatch) Validate(
	ctx context.Context,
	tool serviceports.AgentTool,
	params *serviceports.ToolExecuteParams,
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

func (b *recordBatch) Execute(
	ctx context.Context,
	tool serviceports.AgentTool,
	params *serviceports.ToolExecuteParams,
	run func(ctx context.Context, params serviceports.ToolExecuteParams) error,
) (*agent.ToolExecutionResult, error) {
	if err := guardExecute(tool, *params); err != nil {
		return nil, err
	}
	if !params.ApprovedFromProposal() {
		return nil, b.needsPerson
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
		Action: b.past,
		Kind:   b.nouns,
		Name:   outcome,
	}, nil
}

func (b *recordBatch) outcome(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
	total int,
	failures []batchFailure,
) string {
	outcome := fmt.Sprintf("%d of %d %s", total-len(failures), total, b.past)
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

func (b *recordBatch) failedLabels(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
	failures []batchFailure,
) map[pulid.ID]string {
	if b.labels == nil {
		return map[pulid.ID]string{}
	}

	ids := make([]pulid.ID, 0, len(failures))
	for _, failure := range failures {
		ids = append(ids, failure.id)
	}
	labels := b.labels(ctx, tenantFrom(*params), ids)
	if labels == nil {
		return map[pulid.ID]string{}
	}

	return labels
}
