package insightjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	narrationFirstCheck = 2 * time.Minute
	narrationMaxCheck   = 30 * time.Minute
	// narrationGiveUp is past the batch API's own 24-hour window, so a batch
	// that never ends stops being waited on and its findings keep the
	// detector's wording.
	narrationGiveUp = 26 * time.Hour
)

var narrationOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy:         insightRetryPolicy,
}

// NarrationInput is one organization's narration batch.
type NarrationInput struct {
	temporaljobs.TenantWorkItem

	Pending insightservice.PendingNarration `json:"pending"`
}

// NarrationResult is how many findings the batch worded.
type NarrationResult struct {
	Worded  int  `json:"worded"`
	GaveUp  bool `json:"gaveUp,omitempty"`
	Checked int  `json:"checked"`
}

// startNarration hands the batch to its own workflow, abandoned by the
// refresh, so the sweep finishes while the provider writes.
func startNarration(
	ctx workflow.Context,
	input *OrganizationInsightsInput,
	pending *insightservice.PendingNarration,
) error {
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: fmt.Sprintf(
			"insight-narration/%s/%d", input.OrganizationID, input.Now,
		),
		TaskQueue:         temporaltype.TaskQueueSystem.String(),
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
		Priority:          temporal.Priority{FairnessKey: input.OrganizationID.String()},
	})

	return workflow.ExecuteChildWorkflow(childCtx, InsightNarrationWorkflow, &NarrationInput{
		TenantWorkItem: input.TenantWorkItem,
		Pending:        *pending,
	}).GetChildWorkflowExecution().Get(childCtx, nil)
}

// InsightNarrationWorkflow checks the batch on a widening timer and words the
// findings once it ends.
func InsightNarrationWorkflow(
	ctx workflow.Context,
	input *NarrationInput,
) (*NarrationResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, narrationOptions)
	giveUpAt := time.Unix(input.Pending.Handle.SubmittedAt, 0).Add(narrationGiveUp)
	result := &NarrationResult{}

	wait := narrationFirstCheck
	for {
		if err := workflow.Sleep(ctx, wait); err != nil {
			return result, err
		}
		wait = min(wait*2, narrationMaxCheck)

		var ended bool
		if err := workflow.ExecuteActivity(actx, a.CheckInsightNarrationActivity, input).
			Get(actx, &ended); err != nil {
			return result, err
		}
		result.Checked++
		if ended {
			break
		}
		if workflow.Now(ctx).After(giveUpAt) {
			result.GaveUp = true

			return result, nil
		}
	}

	if err := workflow.ExecuteActivity(actx, a.ApplyInsightNarrationActivity, input).
		Get(actx, &result.Worded); err != nil {
		return result, err
	}

	return result, nil
}

// CheckInsightNarrationActivity reports whether the batch has ended.
func (a *Activities) CheckInsightNarrationActivity(
	ctx context.Context,
	input *NarrationInput,
) (bool, error) {
	return a.insights.NarrationEnded(ctx, input.TenantInfo(), &input.Pending)
}

// ApplyInsightNarrationActivity words the findings the ended batch wrote for.
func (a *Activities) ApplyInsightNarrationActivity(
	ctx context.Context,
	input *NarrationInput,
) (int, error) {
	return a.insights.ApplyNarration(ctx, input.TenantInfo(), &input.Pending)
}
