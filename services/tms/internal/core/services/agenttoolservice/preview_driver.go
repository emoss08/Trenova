package agenttoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

var (
	_ serviceports.ToolPreviewer = (*notifyDriverTool)(nil)
	_ serviceports.ToolPreviewer = (*rejectWorkerPTOTool)(nil)
	_ serviceports.ToolPreviewer = (*cancelWorkerPTOTool)(nil)
	_ serviceports.ToolPreviewer = (*requestCredentialRenewalTool)(nil)
)

// ptoTransitionPreviewer is what the driver-visible PTO decisions preview
// through: the same checks as the decision, and what the driver is sent.
type ptoTransitionPreviewer interface {
	PreviewReject(
		ctx context.Context,
		req *repositories.UpdatePTOStatusRequest,
	) (*serviceports.WorkerPTOTransitionPreview, error)
	PreviewCancel(
		ctx context.Context,
		req *repositories.UpdatePTOStatusRequest,
	) (*serviceports.WorkerPTOTransitionPreview, error)
}

type ptoRejecter interface {
	ptoDecider
	ptoTransitionPreviewer
}

func (t *notifyDriverTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	notice, err := t.drivers.Preview(ctx, request)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would send %s a %s-priority Dash message: %s",
			driverName(notice), notice.Priority, notice.Title),
		driverMessageSend(notice),
	)
	warnDriverUnreachable(preview, notice)
	warnSensitiveContent(preview, notice.Title, notice.Message)

	return preview, nil
}

func (t *rejectWorkerPTOTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	decision, err := t.pto.PreviewReject(ctx, request)
	if err != nil {
		return nil, err
	}

	return ptoDecisionPreview(
		fmt.Sprintf("Would reject %s, telling the driver why: %s",
			ptoLabel(decision.Before), decision.After.RejectionReason),
		decision,
	)
}

func (t *cancelWorkerPTOTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	decision, err := t.pto.PreviewCancel(ctx, request)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would cancel %s, telling the driver why: %s",
		ptoLabel(decision.Before), decision.After.CancellationReason)
	if decision.ReturnsLedger {
		summary += fmt.Sprintf(" The %s booked day(s) go back to the driver's balance.",
			decision.Before.Days.StringFixed(2))
	}

	return ptoDecisionPreview(summary, decision)
}

func (t *requestCredentialRenewalTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	request, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	renewal, err := t.credentials.PreviewRenewal(ctx, request)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(renewal.Credentials))
	for _, credential := range renewal.Credentials {
		names = append(names, credentialLabel(credential))
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would ask %s to renew %s. Nothing on file changes.",
			driverName(renewal.Notification), strings.Join(names, ", ")),
		driverMessageSend(renewal.Notification),
	)
	warnDriverUnreachable(preview, renewal.Notification)
	warnSensitiveContent(preview, request.Note)

	return preview, nil
}

// ptoDecisionPreview is a time-off decision: the request's new status and
// reason, and what the driver is told in Dash and by text.
func ptoDecisionPreview(
	summary string,
	decision *serviceports.WorkerPTOTransitionPreview,
) (*agent.ToolPreview, error) {
	status, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceWorkerPTO,
			ID:       decision.Before.ID,
			Label:    ptoLabel(decision.Before),
			Version:  pinnedVersion(decision.Before.Version),
		},
		decision.Before,
		decision.After,
		toolpreview.WithRefs(map[string]permission.Resource{
			"rejectorId":    permission.ResourceUser,
			"cancelledById": permission.ResourceUser,
		}),
	)
	if err != nil {
		return nil, err
	}

	changes := []*agent.RecordChange{status}
	if decision.Driver != nil {
		changes = append(changes, driverMessageSend(decision.Driver))
	}
	if decision.SMS != nil {
		changes = append(changes, toolpreview.Send(
			toolpreview.Record{
				Resource: permission.ResourceWorker,
				ID:       decision.Before.WorkerID,
				Label:    driverName(decision.Driver),
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
		warnDriverUnreachable(preview, decision.Driver)
	}

	return preview, nil
}

// driverMessageSend is a Dash message to one driver, as the driver would
// read it.
func driverMessageSend(notice *serviceports.DriverNotificationPreview) *agent.RecordChange {
	return toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceWorker,
			ID:       notice.WorkerID,
			Label:    driverName(notice),
		},
		&agent.MessagePreview{
			Channel: agent.MessageChannelDash,
			To:      []string{driverName(notice)},
			Subject: notice.Title,
			Body:    notice.Message,
		},
	)
}

func warnDriverUnreachable(
	preview *agent.ToolPreview,
	notice *serviceports.DriverNotificationPreview,
) {
	if notice == nil || notice.Reachable {
		return
	}

	toolpreview.Warn(preview, agent.PreviewWarningDriverUnreachable,
		driverName(notice)+" has no Dash access, so the message would not reach them.",
		notice.WorkerID.String())
}

func driverName(notice *serviceports.DriverNotificationPreview) string {
	if notice == nil || strings.TrimSpace(notice.WorkerName) == "" {
		return "the driver"
	}

	return notice.WorkerName
}

func ptoLabel(pto *worker.WorkerPTO) string {
	return fmt.Sprintf("%s time off from %s to %s",
		pto.Type,
		time.Unix(pto.StartDate, 0).UTC().Format("2006-01-02"),
		time.Unix(pto.EndDate, 0).UTC().Format("2006-01-02"),
	)
}

func credentialLabel(credential *worker.WorkerCredential) string {
	name := "a credential"
	if credential.CredentialType != nil && credential.CredentialType.Name != "" {
		name = credential.CredentialType.Name
	}
	if credential.ExpiresAt == nil {
		return name
	}

	return name + " (expires " + time.Unix(*credential.ExpiresAt, 0).UTC().Format("2006-01-02") + ")"
}
