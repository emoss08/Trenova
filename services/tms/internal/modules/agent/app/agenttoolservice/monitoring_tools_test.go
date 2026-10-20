package agenttoolservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openFailure() *servicefailure.ServiceFailure {
	return &servicefailure.ServiceFailure{
		ID:         pulid.MustNew("sf_"),
		Number:     "SF-7",
		ShipmentID: pulid.MustNew("shp_"),
		Status:     servicefailure.StatusOpen,
		Version:    3,
	}
}

/*
Detection is the same evaluator the stop actuals run; the tool only asks for
it on one shipment now. It cannot invent a failure, which is why it may run
with a person's approval rather than as a proposal.
*/
func TestEvaluateServiceFailures_RunsTheEvaluatorForTheShipmentAsTheActor(t *testing.T) {
	t.Parallel()

	failures := &agenttoolstest.FakeFailureDecider{}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"force":      true,
	})

	require.NoError(t, operationstools.NewEvaluateServiceFailuresTool(failures).Execute(t.Context(), params))

	require.NotNil(t, failures.Evaluated)
	assert.Equal(t, params.Params["shipmentId"], failures.Evaluated.ShipmentID.String())
	assert.True(t, failures.Evaluated.Force)
	assert.Equal(t, params.OrganizationID, failures.Evaluated.TenantInfo.OrgID)
	assert.Same(t, params.Actor, failures.Actor)
}

func TestResolveServiceFailure_CarriesTheVersionAndTheReason(t *testing.T) {
	t.Parallel()

	existing := openFailure()
	failures := &agenttoolstest.FakeFailureDecider{Existing: existing}
	reasonID := pulid.MustNew("sfrc_")

	require.NoError(t, operationstools.NewResolveServiceFailureTool(failures).Execute(t.Context(), agenttoolstest.ExecuteParams(
		map[string]any{
			"serviceFailureId": existing.ID.String(),
			"reasonCodeId":     reasonID.String(),
			"note":             "  Shipper closed early; driver waited until opening. ",
		},
	)))

	require.NotNil(t, failures.Resolved)
	assert.Equal(t, existing.ID, failures.Resolved.ID)
	assert.Equal(t, existing.ShipmentID, failures.Resolved.ShipmentID)
	assert.Equal(t, reasonID, failures.Resolved.ReasonCodeID)
	assert.EqualValues(t, 3, failures.Resolved.Version, "the read version travels with the write")
	assert.Equal(t, "Shipper closed early; driver waited until opening.", failures.Resolved.Notes)
}

// A failure with no reason code cannot be resolved without one; the service
// would refuse anyway, in words about a lifecycle, so the tool says which
// tool gives the codes.
func TestResolveServiceFailure_InsistsOnAReasonWhenNoneIsOnFile(t *testing.T) {
	t.Parallel()

	existing := openFailure()
	failures := &agenttoolstest.FakeFailureDecider{Existing: existing}

	err := operationstools.NewResolveServiceFailureTool(failures).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"serviceFailureId": existing.ID.String(),
		"note":             "Done.",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exception_list_service_failure_reason_codes")
	assert.Nil(t, failures.Resolved)

	existing.ReasonCodeID = pulid.PtrOrNil(pulid.MustNew("sfrc_"))
	require.NoError(t, operationstools.NewResolveServiceFailureTool(failures).Execute(t.Context(), agenttoolstest.ExecuteParams(
		map[string]any{"serviceFailureId": existing.ID.String(), "note": "Done."},
	)), "a failure that already carries a reason keeps it")
	assert.True(t, failures.Resolved.ReasonCodeID.IsNil())
}

func TestResolveServiceFailure_RefusesATerminalOne(t *testing.T) {
	t.Parallel()

	existing := openFailure()
	existing.Status = servicefailure.StatusResolved
	failures := &agenttoolstest.FakeFailureDecider{Existing: existing}

	err := operationstools.NewResolveServiceFailureTool(failures).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"serviceFailureId": existing.ID.String(),
		"note":             "Again.",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already Resolved")
	assert.Nil(t, failures.Resolved)
}

type fakeDriverNotifier struct {
	sent *drivernotificationservice.DriverNotification
}

func (f *fakeDriverNotifier) Notify(
	_ context.Context,
	req *drivernotificationservice.DriverNotification,
) {
	f.sent = req
}

/*
A driver's message goes through the same template path as every other Dash
notification, so an organization can reword how the office speaks to its
drivers. The tool supplies the title and the text and nothing about how
they are dressed.
*/
func TestNotifyDriver_SendsThroughTheDispatchMessageTemplate(t *testing.T) {
	t.Parallel()

	drivers := &fakeDriverNotifier{}
	workerID := pulid.MustNew("wrk_")
	shipmentID := pulid.MustNew("shp_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		"workerId":   workerID.String(),
		"title":      "Delivery moved to 3 PM",
		"message":    "Houston DC moved your appointment to 3 PM. No need to rush.",
		"priority":   "high",
		"shipmentId": shipmentID.String(),
	})

	require.NoError(t, operationstools.NewNotifyDriverTool(drivers).Execute(t.Context(), params))

	require.NotNil(t, drivers.sent)
	assert.Equal(t, workerID, drivers.sent.WorkerID)
	assert.Equal(t, operationstools.DispatchMessageEvent, drivers.sent.EventType)
	assert.Equal(t, notification.PriorityHigh, drivers.sent.Priority)
	assert.Equal(t, params.OrganizationID, drivers.sent.TenantInfo.OrgID)
	assert.Equal(t, "/loads/"+shipmentID.String(), drivers.sent.Link)

	context, ok := drivers.sent.Context.(documenttemplate.DriverNotificationContext)
	require.True(t, ok)
	assert.Equal(t, "Delivery moved to 3 PM", context.AlertTitle)
	assert.Equal(
		t,
		"Houston DC moved your appointment to 3 PM. No need to rush.",
		context.AlertMessage,
	)
}

func TestNotifyDriver_DefaultsThePriorityAndBoundsTheMessage(t *testing.T) {
	t.Parallel()

	drivers := &fakeDriverNotifier{}
	tool := operationstools.NewNotifyDriverTool(drivers)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"title":    "Call dispatch",
		"message":  "Call when you are parked.",
	})))
	assert.Equal(t, notification.PriorityMedium, drivers.sent.Priority)

	drivers.sent = nil
	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"title":    "Long",
		"message":  strings.Repeat("x", operationstools.MaxDriverMessageChars+1),
	}))
	require.Error(t, err)
	assert.Nil(t, drivers.sent)

	err = tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"title":    "Urgent",
		"message":  "Now.",
		"priority": "urgent",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "priority")
}

/*
The customer is written to at the addresses on the customer's own record, in
the organization's own letterhead, and the shipment's thread records that it
happened. The model supplies the words and nothing else about where they go.
*/
func TestEmailCustomer_WritesToTheCustomersContactsAndRecordsIt(t *testing.T) {
	t.Parallel()

	tool, mailer, customers, comments, renderer := agenttoolstest.CustomerEmailFixture()
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 running about an hour late",
		"body":       "The truck is held in traffic; we now expect 3:30 PM.",
	})
	params.IdempotencyKey = "idem-1"

	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, mailer.Sent)
	assert.Equal(t, []string{"ops@acme.example", "ap@acme.example"}, mailer.Sent.To)
	assert.Equal(t, email.PurposeOperations, mailer.Sent.Purpose)
	assert.Equal(t, "[Acme] S12345 running about an hour late", mailer.Sent.Subject)
	assert.Equal(t, "idem-1", mailer.Sent.IdempotencyKey)
	assert.True(t, customers.Req.IncludeEmailProfile)

	require.NotNil(t, renderer.Req)
	assert.Equal(t, documenttemplate.KindAgentCustomerUpdateEmail, renderer.Req.Kind)
	context, ok := renderer.Req.Data.(documenttemplate.AgentEmailContext)
	require.True(t, ok)
	assert.Equal(t, "Acme Freight", context.CustomerName)
	assert.Equal(t, "S12345", context.ShipmentProNumber)

	require.NotNil(t, comments.Created)
	assert.Equal(t, shipment.CommentTypeCustomerUpdate, comments.Created.Type)
	assert.Contains(t, comments.Created.Comment, "ops@acme.example")
	assert.Contains(t, comments.Created.Comment, "we now expect 3:30 PM")
}

func TestEmailCustomer_RefusesACustomerWithNoContacts(t *testing.T) {
	t.Parallel()

	tool, mailer, customers, _, _ := agenttoolstest.CustomerEmailFixture()
	customers.Entity.EmailProfile = nil
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "Update",
		"body":       "Body.",
	})
	params.IdempotencyKey = "idem-3"

	err := tool.Execute(t.Context(), params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no notice recipients")
	assert.Nil(t, mailer.Sent)
}

func TestEmailCustomer_IsAProposalThatNeedsAnIdempotencyKey(t *testing.T) {
	t.Parallel()

	tool, mailer, _, _, _ := agenttoolstest.CustomerEmailFixture()
	assert.Equal(t, agent.TierPropose, tool.Policy().DefaultTier)
	assert.True(t, tool.Policy().Idempotent)
	assert.Equal(t, permission.ResourceCustomerCommunication, tool.Policy().Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "Update",
		"body":       "Body.",
	}))

	require.ErrorIs(t, err, common.ErrMissingIdempotencyKey)
	assert.Nil(t, mailer.Sent)
}

func TestEmailCustomer_ReportsARecordFailureAfterTheSend(t *testing.T) {
	t.Parallel()

	tool, mailer, _, _, _ := agenttoolstest.CustomerEmailFixture()
	tool.Deps.Comments = &failingCommentWriter{}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "Update",
		"body":       "Body.",
	})
	params.IdempotencyKey = "idem-2"

	err := tool.Execute(t.Context(), params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "was sent but could not be recorded")
	assert.NotNil(t, mailer.Sent, "the send is not undone by a failed record")
}

type failingCommentWriter struct {
	shipmentports.ShipmentCommentService
}

// Nothing has gone to this customer yet; the failure under test is the
// record after the send, not the guard before it.
func (f *failingCommentWriter) ListByShipmentID(
	context.Context,
	*shipmentports.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return &pagination.CursorListResult[*shipment.ShipmentComment]{}, nil
}

func (failingCommentWriter) CreateSystem(
	context.Context,
	*shipmentports.CreateSystemShipmentCommentRequest,
) (*shipment.ShipmentComment, error) {
	return nil, errors.New("thread locked")
}

func TestSendDetentionNotice_SendsAsTheActorNotTheSweep(t *testing.T) {
	t.Parallel()

	det := &agenttoolstest.FakeDetention{}
	occurrenceID := pulid.MustNew("dto_")
	params := agenttoolstest.ExecuteParams(map[string]any{"occurrenceId": occurrenceID.String()})

	require.NoError(t, operationstools.NewSendDetentionNoticeTool(det).Execute(t.Context(), params))

	require.NotNil(t, det.Noticed)
	assert.Equal(t, occurrenceID, det.Noticed.OccurrenceID)
	assert.Equal(t, params.Actor.UserID, det.Noticed.UserID)
	assert.False(t, det.Noticed.Automatic)
}

func TestWaiveDetention_RequiresACodedReasonAndANote(t *testing.T) {
	t.Parallel()

	det := &agenttoolstest.FakeDetention{}
	tool := operationstools.NewWaiveDetentionTool(det)
	occurrenceID := pulid.MustNew("dto_")

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(map[string]any{
		"occurrenceId": occurrenceID.String(),
		"reason":       "CarrierFault",
		"note":         " Our truck arrived two hours late. ",
	})))
	require.NotNil(t, det.Waived)
	assert.Equal(t, detention.WaiverReasonCarrierFault, det.Waived.Reason)
	assert.Equal(t, "Our truck arrived two hours late.", det.Waived.Note)

	det.Waived = nil
	err := tool.Execute(t.Context(), agenttoolstest.ApprovedParams(map[string]any{
		"occurrenceId": occurrenceID.String(),
		"reason":       "BecauseISaidSo",
		"note":         "x",
	}))
	require.Error(t, err)
	assert.Nil(t, det.Waived)
	assert.Equal(t, agent.TierPropose, tool.Policy().DefaultTier)
}

func TestWaiveDetention_GivesUpRevenueOnlyOnceAPersonApproves(t *testing.T) {
	t.Parallel()

	det := &agenttoolstest.FakeDetention{}
	tool := operationstools.NewWaiveDetentionTool(det)

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"occurrenceId": pulid.MustNew("dto_").String(),
		"reason":       "CarrierFault",
		"note":         "Our truck arrived two hours late.",
	}))
	require.ErrorIs(t, err, common.ErrApprovalNeedsAPerson)
	assert.Nil(t, det.Waived)

	policy := tool.Policy()
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
}

func TestMonitoringActionTools_RejectAMismatchedActor(t *testing.T) {
	t.Parallel()

	tool, _, _, _, _ := agenttoolstest.CustomerEmailFixture()
	for _, candidate := range []agentports.AgentTool{
		operationstools.NewEvaluateServiceFailuresTool(&agenttoolstest.FakeFailureDecider{}),
		operationstools.NewResolveServiceFailureTool(&agenttoolstest.FakeFailureDecider{}),
		operationstools.NewNotifyDriverTool(&fakeDriverNotifier{}),
		tool,
		operationstools.NewSendDetentionNoticeTool(&agenttoolstest.FakeDetention{}),
		operationstools.NewWaiveDetentionTool(&agenttoolstest.FakeDetention{}),
	} {
		params := agenttoolstest.ExecuteParams(map[string]any{})
		params.Actor.BusinessUnitID = pulid.MustNew("bu_")
		require.ErrorIs(
			t,
			candidate.Execute(t.Context(), params),
			common.ErrTenantMismatch,
			candidate.Name(),
		)
	}
}

// Reaching outside the organization is its own permission. A dispatcher who
// may edit a shipment does not thereby get to email its customer, and one
// who may read a driver's record does not thereby get to page their phone:
// each outbound tool is gated on the communication it sends, not on the
// record it is about.
func TestOutboundTools_AreGatedOnTheCommunicationTheySend(t *testing.T) {
	t.Parallel()

	customerTool, _, _, _, _ := agenttoolstest.CustomerEmailFixture()
	tools := []struct {
		tool     agentports.AgentTool
		resource permission.Resource
	}{
		{operationstools.NewNotifyDriverTool(&fakeDriverNotifier{}), permission.ResourceDriverMessage},
		{customerTool, permission.ResourceCustomerCommunication},
		{operationstools.NewSendDetentionNoticeTool(&agenttoolstest.FakeDetention{}), permission.ResourceCustomerCommunication},
	}
	for _, entry := range tools {
		assert.Equal(t, entry.resource, entry.tool.Policy().Resource, entry.tool.Name())
		assert.Equal(t, permission.OpCreate, entry.tool.Policy().Operation, entry.tool.Name())
		assert.True(
			t,
			permission.IsAgentAllowed(
				entry.tool.Policy().Resource,
				entry.tool.Policy().Operation,
			),
			"%s must be reachable by an agent principal",
			entry.tool.Name(),
		)
	}
}

/*
The customer update desk is woken by every arrival and every departure, so a
shipment crossing a yard can raise several runs within a few minutes. The
rule against telling the customer twice used to be a line in the prompt,
which held exactly as well as the model's attention; the comment the send
leaves is the record, and reading it back is what makes the rule a rule.
*/
func TestEmailCustomer_WillNotTellTheSameCustomerTwiceWithinTheHour(t *testing.T) {
	t.Parallel()

	tool, mailer, _, comments, _ := agenttoolstest.CustomerEmailFixture()
	comments.Existing = []*shipment.ShipmentComment{{
		CreatedAt: timeutils.NowUnix() - 600,
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "agent", "tool": "customer_send_email"},
	}}

	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 has departed",
		"body":       "The truck left the yard at 3:32 PM.",
	})
	params.IdempotencyKey = "idem-2"

	err := tool.Execute(t.Context(), params)
	require.ErrorIs(t, err, common.ErrCustomerAlreadyTold)
	assert.Nil(t, mailer.Sent, "nothing may go out once the customer has been told")
	assert.Nil(t, comments.Created)
}

// A dispatcher typing an update on the shipment is not an email to the
// customer, and must not silence one.
func TestEmailCustomer_ATypedCommentDoesNotSilenceTheDesk(t *testing.T) {
	t.Parallel()

	tool, mailer, _, comments, _ := agenttoolstest.CustomerEmailFixture()
	comments.Existing = []*shipment.ShipmentComment{{
		CreatedAt: timeutils.NowUnix() - 60,
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "user"},
	}}

	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 has departed",
		"body":       "The truck left the yard at 3:32 PM.",
	})
	params.IdempotencyKey = "idem-3"

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, mailer.Sent)
}
