package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fakes below answer a preview from the request alone, so what a preview
// shows can be compared with what the write then sends, and a fake that
// recorded a send during a preview would show it.

func (f *fakeDriverNotifier) Preview(
	_ context.Context,
	req *drivernotificationservice.DriverNotification,
) (*serviceports.DriverNotificationPreview, error) {
	return renderedDriverNotice(req, true), nil
}

func renderedDriverNotice(
	req *drivernotificationservice.DriverNotification,
	reachable bool,
) *serviceports.DriverNotificationPreview {
	data, _ := req.Context.(documenttemplate.DriverNotificationContext)

	return &serviceports.DriverNotificationPreview{
		WorkerID:   req.WorkerID,
		WorkerName: "Marcus Dell",
		Reachable:  reachable,
		Title:      data.AlertTitle,
		Message:    data.AlertMessage,
		Priority:   req.Priority,
		Link:       req.Link,
	}
}

type unreachableDriverNotifier struct{ fakeDriverNotifier }

func (f *unreachableDriverNotifier) Preview(
	_ context.Context,
	req *drivernotificationservice.DriverNotification,
) (*serviceports.DriverNotificationPreview, error) {
	return renderedDriverNotice(req, false), nil
}

func (f *fakePTODecider) PreviewReject(
	_ context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*serviceports.WorkerPTOTransitionPreview, error) {
	return decidedPTO(req, worker.PTOStatusRequested, worker.PTOStatusRejected), nil
}

func (f *fakePTODecider) PreviewCancel(
	_ context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*serviceports.WorkerPTOTransitionPreview, error) {
	return decidedPTO(req, worker.PTOStatusApproved, worker.PTOStatusCancelled), nil
}

func decidedPTO(
	req *repositories.UpdatePTOStatusRequest,
	from, to worker.PTOStatus,
) *serviceports.WorkerPTOTransitionPreview {
	before := &worker.WorkerPTO{
		ID:        req.ID,
		WorkerID:  pulid.MustNew("wrk_"),
		Status:    from,
		Type:      worker.PTOTypeVacation,
		StartDate: 1_767_225_600,
		EndDate:   1_767_398_400,
		Days:      decimal.NewFromInt(2),
		Version:   3,
	}
	after := *before
	after.Status = to
	if to == worker.PTOStatusRejected {
		after.RejectorID = req.UserID
		after.RejectionReason = req.Reason
	} else {
		after.CancelledByID = req.UserID
		after.CancellationReason = req.Reason
	}

	return &serviceports.WorkerPTOTransitionPreview{
		Before:        before,
		After:         &after,
		ReturnsLedger: from == worker.PTOStatusApproved,
		Driver: &serviceports.DriverNotificationPreview{
			WorkerID:   before.WorkerID,
			WorkerName: "Marcus Dell",
			Title:      "Your time off was declined",
			Message:    req.Reason,
		},
		SMS: &serviceports.DriverSMSPreview{
			PhoneNumber: "+15555550100",
			Message:     "Dana Ortiz has rejected your PTO request. Reason: " + req.Reason,
		},
	}
}

func (s *stubCredentialActor) PreviewRenewal(
	_ context.Context,
	req *workercredentialservice.RenewalRequest,
) (*workercredentialservice.RenewalPreview, error) {
	expires := int64(1_790_000_000)

	return &workercredentialservice.RenewalPreview{
		Credentials: []*worker.WorkerCredential{{
			ID:             req.CredentialIDs[0],
			ExpiresAt:      &expires,
			CredentialType: &worker.WorkerCredentialType{Name: "CDL"},
		}},
		Notification: &serviceports.DriverNotificationPreview{
			WorkerID:   req.WorkerID,
			WorkerName: "Marcus Dell",
			Reachable:  false,
			Title:      "Renew your CDL",
			Message:    req.Note,
		},
	}, nil
}

func TestNotifyDriverPreview_ShowsTheDashMessage(t *testing.T) {
	t.Parallel()

	drivers := &fakeDriverNotifier{}
	tool := newNotifyDriverTool(drivers)
	params := executeParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"title":    "Delivery moved to 3 PM",
		"message":  "Houston DC moved your appointment to 3 PM.",
		"priority": "high",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, drivers.sent, "a preview must not message the driver")
	assert.Empty(t, preview.Warnings)

	send := findChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, permission.ResourceWorker, send.Resource)
	require.NotNil(t, send.Message)
	assert.Equal(t, agent.MessageChannelDash, send.Message.Channel)
	assert.Equal(t, []string{"Marcus Dell"}, send.Message.To)

	require.NoError(t, tool.Execute(t.Context(), params))
	sent := renderedDriverNotice(drivers.sent, true)
	assert.Equal(t, sent.Title, send.Message.Subject)
	assert.Equal(t, sent.Message, send.Message.Body)
}

func TestNotifyDriverPreview_WarnsADriverWithoutDash(t *testing.T) {
	t.Parallel()

	tool := newNotifyDriverTool(&unreachableDriverNotifier{})
	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), executeParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"title":    "Call dispatch",
		"message":  "Call when you are parked.",
	}))
	require.NoError(t, err)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agent.PreviewWarningDriverUnreachable, preview.Warnings[0].Code)
}

func TestRejectWorkerPTOPreview_ShowsTheDecisionAndWhatTheDriverIsSent(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	tool := newRejectWorkerPTOTool(pto)
	params := executeParams(map[string]any{
		"ptoId":  pulid.MustNew("wrkpto_").String(),
		"reason": "Coverage gap on that route",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, pto.rejected, "a preview must not reject")
	assert.Contains(t, preview.Summary, "Coverage gap on that route")

	status := findChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, permission.ResourceWorkerPTO, status.Resource)
	assert.Equal(t, "Rejected", findField(t, status, "status").After)
	assert.Equal(t, "Coverage gap on that route", findField(t, status, "rejectionReason").After)

	var channels []agent.MessageChannel
	for i := range preview.Changes {
		if preview.Changes[i].Message != nil {
			channels = append(channels, preview.Changes[i].Message.Channel)
		}
	}
	assert.Equal(t, []agent.MessageChannel{agent.MessageChannelDash, agent.MessageChannelSMS}, channels)
	assert.Equal(t, agent.PreviewWarningDriverUnreachable, preview.Warnings[0].Code)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Coverage gap on that route", pto.rejected.Reason)
}

func TestCancelWorkerPTOPreview_SaysTheDaysGoBack(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	tool := newCancelWorkerPTOTool(pto)
	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), executeParams(map[string]any{
		"ptoId":  pulid.MustNew("wrkpto_").String(),
		"reason": "Route covered",
	}))
	require.NoError(t, err)
	require.Nil(t, pto.cancelled)
	assert.Contains(t, preview.Summary, "2.00 booked day(s) go back")
}

func TestRequestCredentialRenewalPreview_ShowsTheAskAndWhoCannotReadIt(t *testing.T) {
	t.Parallel()

	stub := &stubCredentialActor{}
	tool := newRequestCredentialRenewalTool(stub)
	params := deskParams(map[string]any{
		"workerId":      pulid.MustNew("wrk_").String(),
		"credentialIds": []any{pulid.MustNew("wcred_").String()},
		"note":          "Your CDL is due this month.",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, stub.asked, "a preview must not ask")
	assert.Contains(t, preview.Summary, "CDL (expires 2026-09-21)")

	send := findChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, "Renew your CDL", send.Message.Subject)
	assert.Equal(t, "Your CDL is due this month.", send.Message.Body)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agent.PreviewWarningDriverUnreachable, preview.Warnings[0].Code)
}
