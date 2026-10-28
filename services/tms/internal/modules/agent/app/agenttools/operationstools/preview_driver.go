package operationstools

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/shared/timeutils"
)

// ptoTransitionPreviewer is what the driver-visible PTO decisions preview
// through: the same checks as the decision, and what the driver is sent.
type ptoTransitionPreviewer interface {
	PreviewReject(
		ctx context.Context,
		req *workerports.UpdatePTOStatusRequest,
	) (*workerports.WorkerPTOTransitionPreview, error)
	PreviewCancel(
		ctx context.Context,
		req *workerports.UpdatePTOStatusRequest,
	) (*workerports.WorkerPTOTransitionPreview, error)
}

type ptoRejecter interface {
	common.PtoDecider
	ptoTransitionPreviewer
}

func (t *NotifyDriverTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	notice, err := t.drivers.Preview(ctx, request)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would send %s a %s-priority Dash message: %s",
			common.DriverName(notice), notice.Priority, notice.Title),
		common.DriverMessageSend(notice),
	)
	common.WarnDriverUnreachable(preview, notice)
	common.WarnSensitiveContent(preview, notice.Title, notice.Message)

	return preview, nil
}

func (t *RejectWorkerPTOTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	decision, err := t.pto.PreviewReject(ctx, request)
	if err != nil {
		return nil, err
	}

	return ptoDecisionPreview(
		fmt.Sprintf("Would reject %s, telling the driver why: %s",
			PtoLabel(decision.Before, decision.Location), decision.After.RejectionReason),
		decision,
	)
}

func (t *CancelWorkerPTOTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	decision, err := t.pto.PreviewCancel(ctx, request)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would cancel %s, telling the driver why: %s",
		PtoLabel(decision.Before, decision.Location), decision.After.CancellationReason)
	if decision.ReturnsLedger {
		summary += fmt.Sprintf(" The %s booked day(s) go back to the driver's balance.",
			decision.Before.Days.StringFixed(2))
	}

	return ptoDecisionPreview(summary, decision)
}

// ptoDecisionPreview is a time-off decision: the request's new status and
// reason, and what the driver is told in Dash and by text.
func PtoDecisionOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.WithRefs(map[string]permission.Resource{
			"approverId":    permission.ResourceUser,
			"rejectorId":    permission.ResourceUser,
			"cancelledById": permission.ResourceUser,
		}),
	}
}

func ptoDecisionPreview(
	summary string,
	decision *workerports.WorkerPTOTransitionPreview,
) (*agent.ToolPreview, error) {
	status, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceWorkerPTO,
			ID:       decision.Before.ID,
			Label:    PtoLabel(decision.Before, decision.Location),
			Version:  common.PinnedVersion(decision.Before.Version),
		},
		decision.Before,
		decision.After,
		PtoDecisionOptions()...,
	)
	if err != nil {
		return nil, err
	}

	changes := []*agent.RecordChange{status}
	if decision.Driver != nil {
		changes = append(changes, common.DriverMessageSend(decision.Driver))
	}
	if decision.SMS != nil {
		changes = append(changes, toolpreview.Send(
			toolpreview.Record{
				Resource: permission.ResourceWorker,
				ID:       decision.Before.WorkerID,
				Label:    common.DriverName(decision.Driver),
			},
			&agent.MessagePreview{
				Channel: agent.MessageChannelSMS,
				To:      []string{decision.SMS.PhoneNumber},
				Body:    decision.SMS.Message,
			},
		))
	}

	preview := toolpreview.Build(summary, changes...)
	if decision.Driver != nil {
		common.WarnDriverUnreachable(preview, decision.Driver)
	}

	return preview, nil
}

// ptoLabel names a request by its days in the organization's zone, the one
// they are booked in. In UTC a request stored in the evening read a day later
// than worker_list_time_off showed it, and an agent withdrew six correct approvals
// as mismatched. The zone is the organization's, not the viewer's, so the
// preview's digest is the same whoever computes it.
func PtoLabel(pto *worker.WorkerPTO, loc *time.Location) string {
	return fmt.Sprintf("%s time off from %s to %s",
		pto.Type,
		timeutils.FormatCalendarDate(pto.StartDate, loc),
		timeutils.FormatCalendarDate(pto.EndDate, loc),
	)
}
