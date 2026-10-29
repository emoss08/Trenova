package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/findingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ptodecisiontools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *fakeDriverNotifier) Preview(
	_ context.Context,
	req *drivernotificationservice.DriverNotification,
) (*notificationports.DriverNotificationPreview, error) {
	return renderedDriverNotice(req, true), nil
}

func renderedDriverNotice(
	req *drivernotificationservice.DriverNotification,
	reachable bool,
) *notificationports.DriverNotificationPreview {
	data, _ := req.Context.(notificationtemplate.DriverNotificationContext)

	return &notificationports.DriverNotificationPreview{
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
) (*notificationports.DriverNotificationPreview, error) {
	return renderedDriverNotice(req, false), nil
}

func (f *fakePTODecider) PreviewReject(
	_ context.Context,
	req *workerptoports.UpdatePTOStatusRequest,
) (*workerptoports.WorkerPTOTransitionPreview, error) {
	return decidedPTO(req, worker.PTOStatusRequested, worker.PTOStatusRejected), nil
}

func (f *fakePTODecider) PreviewCancel(
	_ context.Context,
	req *workerptoports.UpdatePTOStatusRequest,
) (*workerptoports.WorkerPTOTransitionPreview, error) {
	return decidedPTO(req, worker.PTOStatusApproved, worker.PTOStatusCancelled), nil
}

func decidedPTO(
	req *workerptoports.UpdatePTOStatusRequest,
	from, to worker.PTOStatus,
) *workerptoports.WorkerPTOTransitionPreview {
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

	return &workerptoports.WorkerPTOTransitionPreview{
		Before:        before,
		After:         &after,
		ReturnsLedger: from == worker.PTOStatusApproved,
		Driver: &notificationports.DriverNotificationPreview{
			WorkerID:   before.WorkerID,
			WorkerName: "Marcus Dell",
			Title:      "Your time off was declined",
			Message:    req.Reason,
		},
		SMS: &notificationports.DriverSMSPreview{
			PhoneNumber: "+15555550100",
			Message:     "Dana Ortiz has rejected your PTO request. Reason: " + req.Reason,
		},
	}
}

func TestNotifyDriverPreview_ShowsTheDashMessage(t *testing.T) {
	t.Parallel()

	drivers := &fakeDriverNotifier{}
	tool := operationstools.NewNotifyDriverTool(drivers)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"title":    "Delivery moved to 3 PM",
		"message":  "Houston DC moved your appointment to 3 PM.",
		"priority": "high",
	})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, drivers.sent, "a preview must not message the driver")
	assert.Empty(t, preview.Warnings)

	send := agenttoolstest.FindChange(t, preview, agentpreview.PreviewOperationSend)
	assert.Equal(t, permission.ResourceWorker, send.Resource)
	require.NotNil(t, send.Message)
	assert.Equal(t, agentpreview.MessageChannelDash, send.Message.Channel)
	assert.Equal(t, []string{"Marcus Dell"}, send.Message.To)

	require.NoError(t, tool.Execute(t.Context(), params))
	sent := renderedDriverNotice(drivers.sent, true)
	assert.Equal(t, sent.Title, send.Message.Subject)
	assert.Equal(t, sent.Message, send.Message.Body)
}

func TestNotifyDriverPreview_WarnsADriverWithoutDash(t *testing.T) {
	t.Parallel()

	tool := operationstools.NewNotifyDriverTool(&unreachableDriverNotifier{})
	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(
		t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			"workerId": pulid.MustNew("wrk_").String(),
			"title":    "Call dispatch",
			"message":  "Call when you are parked.",
		}),
	)
	require.NoError(t, err)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agentpreview.PreviewWarningDriverUnreachable, preview.Warnings[0].Code)
}

func TestRejectWorkerPTOPreview_ShowsTheDecisionAndWhatTheDriverIsSent(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	tool := ptodecisiontools.NewRejectWorkerPTOTool(pto)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"ptoId":  pulid.MustNew("wrkpto_").String(),
		"reason": "Coverage gap on that route",
	})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, pto.rejected, "a preview must not reject")
	assert.Contains(t, preview.Summary, "Coverage gap on that route")

	status := agenttoolstest.FindChange(t, preview, agentpreview.PreviewOperationUpdate)
	assert.Equal(t, permission.ResourceWorkerPTO, status.Resource)
	assert.Equal(t, "Rejected", agenttoolstest.FindField(t, status, "status").After)
	assert.Equal(
		t,
		"Coverage gap on that route",
		agenttoolstest.FindField(t, status, "rejectionReason").After,
	)

	var channels []agentpreview.MessageChannel
	for i := range preview.Changes {
		if preview.Changes[i].Message != nil {
			channels = append(channels, preview.Changes[i].Message.Channel)
		}
	}
	assert.Equal(
		t,
		[]agentpreview.MessageChannel{
			agentpreview.MessageChannelDash,
			agentpreview.MessageChannelSMS,
		},
		channels,
	)
	assert.Equal(t, agentpreview.PreviewWarningDriverUnreachable, preview.Warnings[0].Code)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Coverage gap on that route", pto.rejected.Reason)
}

func TestCancelWorkerPTOPreview_SaysTheDaysGoBack(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	tool := ptodecisiontools.NewCancelWorkerPTOTool(pto)
	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(
		t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			"ptoId":  pulid.MustNew("wrkpto_").String(),
			"reason": "Route covered",
		}),
	)
	require.NoError(t, err)
	require.Nil(t, pto.cancelled)
	assert.Contains(t, preview.Summary, "2.00 booked day(s) go back")
}

func TestRequestCredentialRenewalPreview_ShowsTheAskAndWhoCannotReadIt(t *testing.T) {
	t.Parallel()

	stub := &agenttoolstest.StubCredentialActor{}
	tool := findingtools.NewRequestCredentialRenewalTool(stub)
	params := agenttoolstest.DeskParams(map[string]any{
		"workerId":      pulid.MustNew("wrk_").String(),
		"credentialIds": []any{pulid.MustNew("wcred_").String()},
		"note":          "Your CDL is due this month.",
	})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, stub.Asked, "a preview must not ask")
	assert.Contains(t, preview.Summary, "CDL (expires 2026-09-21)")

	send := agenttoolstest.FindChange(t, preview, agentpreview.PreviewOperationSend)
	assert.Equal(t, "Renew your CDL", send.Message.Subject)
	assert.Equal(t, "Your CDL is due this month.", send.Message.Body)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agentpreview.PreviewWarningDriverUnreachable, preview.Warnings[0].Code)
}

/*
A request stored at 01:17 UTC is the evening before in Chicago, the zone its
days are booked in. Labelled in UTC, every request read a day later than
worker_list_time_off showed it, and gpt-6-luna withdrew six correct approvals as
mismatched.
*/
func TestPTOLabel_NamesTheDaysInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	start := time.Date(2027, 5, 3, 1, 17, 50, 0, time.UTC).Unix()
	pto := &worker.WorkerPTO{
		Type:      worker.PTOTypeVacation,
		StartDate: start,
		EndDate:   start + 86_400,
	}

	assert.Equal(
		t,
		"Vacation time off from 2027-05-02 to 2027-05-03",
		ptodecisiontools.PtoLabel(pto, chicago),
	)
	assert.Equal(
		t,
		"Vacation time off from 2027-05-03 to 2027-05-04",
		ptodecisiontools.PtoLabel(pto, nil),
		"with no zone known it falls back to UTC",
	)
}
