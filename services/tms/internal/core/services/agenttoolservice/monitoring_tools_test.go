package agenttoolservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFailureDecider struct {
	existing  *servicefailure.ServiceFailure
	evaluated *serviceports.EvaluateShipmentServiceFailuresRequest
	resolved  *serviceports.ServiceFailureLifecycleRequest
	actor     *serviceports.RequestActor
}

func (f *fakeFailureDecider) EvaluateShipment(
	_ context.Context,
	req *serviceports.EvaluateShipmentServiceFailuresRequest,
	actor *serviceports.RequestActor,
) (*serviceports.ServiceFailureEvaluationResult, error) {
	f.evaluated = req
	f.actor = actor

	return &serviceports.ServiceFailureEvaluationResult{}, nil
}

func (f *fakeFailureDecider) GetByID(
	_ context.Context,
	req *repositories.GetServiceFailureByIDRequest,
) (*servicefailure.ServiceFailure, error) {
	if f.existing != nil && f.existing.ID == req.ID {
		return f.existing, nil
	}

	return nil, errortypes.NewNotFoundError("ServiceFailure not found")
}

func (f *fakeFailureDecider) Resolve(
	_ context.Context,
	req *serviceports.ServiceFailureLifecycleRequest,
	actor *serviceports.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	f.resolved = req
	f.actor = actor

	return f.existing, nil
}

func (f *fakeFailureDecider) PreviewEvaluateShipment(
	context.Context,
	*serviceports.EvaluateShipmentServiceFailuresRequest,
) (*serviceports.ServiceFailureDetectionPlan, error) {
	return &serviceports.ServiceFailureDetectionPlan{}, nil
}

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

	failures := &fakeFailureDecider{}
	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"force":      true,
	})

	require.NoError(t, newEvaluateServiceFailuresTool(failures).Execute(t.Context(), params))

	require.NotNil(t, failures.evaluated)
	assert.Equal(t, params.Params["shipmentId"], failures.evaluated.ShipmentID.String())
	assert.True(t, failures.evaluated.Force)
	assert.Equal(t, params.OrganizationID, failures.evaluated.TenantInfo.OrgID)
	assert.Same(t, params.Actor, failures.actor)
}

func TestResolveServiceFailure_CarriesTheVersionAndTheReason(t *testing.T) {
	t.Parallel()

	existing := openFailure()
	failures := &fakeFailureDecider{existing: existing}
	reasonID := pulid.MustNew("sfrc_")

	require.NoError(t, newResolveServiceFailureTool(failures).Execute(t.Context(), executeParams(
		map[string]any{
			"serviceFailureId": existing.ID.String(),
			"reasonCodeId":     reasonID.String(),
			"notes":            "  Shipper closed early; driver waited until opening. ",
		},
	)))

	require.NotNil(t, failures.resolved)
	assert.Equal(t, existing.ID, failures.resolved.ID)
	assert.Equal(t, existing.ShipmentID, failures.resolved.ShipmentID)
	assert.Equal(t, reasonID, failures.resolved.ReasonCodeID)
	assert.EqualValues(t, 3, failures.resolved.Version, "the read version travels with the write")
	assert.Equal(t, "Shipper closed early; driver waited until opening.", failures.resolved.Notes)
}

// A failure with no reason code cannot be resolved without one; the service
// would refuse anyway, in words about a lifecycle, so the tool says which
// tool gives the codes.
func TestResolveServiceFailure_InsistsOnAReasonWhenNoneIsOnFile(t *testing.T) {
	t.Parallel()

	existing := openFailure()
	failures := &fakeFailureDecider{existing: existing}

	err := newResolveServiceFailureTool(failures).Execute(t.Context(), executeParams(map[string]any{
		"serviceFailureId": existing.ID.String(),
		"notes":            "Done.",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list_service_failure_reason_codes")
	assert.Nil(t, failures.resolved)

	existing.ReasonCodeID = pulid.PtrOrNil(pulid.MustNew("sfrc_"))
	require.NoError(t, newResolveServiceFailureTool(failures).Execute(t.Context(), executeParams(
		map[string]any{"serviceFailureId": existing.ID.String(), "notes": "Done."},
	)), "a failure that already carries a reason keeps it")
	assert.True(t, failures.resolved.ReasonCodeID.IsNil())
}

func TestResolveServiceFailure_RefusesATerminalOne(t *testing.T) {
	t.Parallel()

	existing := openFailure()
	existing.Status = servicefailure.StatusResolved
	failures := &fakeFailureDecider{existing: existing}

	err := newResolveServiceFailureTool(failures).Execute(t.Context(), executeParams(map[string]any{
		"serviceFailureId": existing.ID.String(),
		"notes":            "Again.",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already Resolved")
	assert.Nil(t, failures.resolved)
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
	params := executeParams(map[string]any{
		"workerId":   workerID.String(),
		"title":      "Delivery moved to 3 PM",
		"message":    "Houston DC moved your appointment to 3 PM. No need to rush.",
		"priority":   "high",
		"shipmentId": shipmentID.String(),
	})

	require.NoError(t, newNotifyDriverTool(drivers).Execute(t.Context(), params))

	require.NotNil(t, drivers.sent)
	assert.Equal(t, workerID, drivers.sent.WorkerID)
	assert.Equal(t, dispatchMessageEvent, drivers.sent.EventType)
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
	tool := newNotifyDriverTool(drivers)

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"title":    "Call dispatch",
		"message":  "Call when you are parked.",
	})))
	assert.Equal(t, notification.PriorityMedium, drivers.sent.Priority)

	drivers.sent = nil
	err := tool.Execute(t.Context(), executeParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"title":    "Long",
		"message":  strings.Repeat("x", maxDriverMessageChars+1),
	}))
	require.Error(t, err)
	assert.Nil(t, drivers.sent)

	err = tool.Execute(t.Context(), executeParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"title":    "Urgent",
		"message":  "Now.",
		"priority": "urgent",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "priority")
}

type fakeCustomerRepo struct {
	repositories.CustomerRepository

	entity *customer.Customer
	req    repositories.GetCustomerByIDRequest
}

func (f *fakeCustomerRepo) GetByID(
	_ context.Context,
	req repositories.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	f.req = req

	return f.entity, nil
}

type fakeShipmentReader struct {
	repositories.ShipmentRepository

	entity *shipment.Shipment
}

func (f *fakeShipmentReader) GetByID(
	_ context.Context,
	_ *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return f.entity, nil
}

type fakeMailer struct {
	sent *serviceports.SendEmailRequest
	err  error
}

func (f *fakeMailer) Send(
	_ context.Context,
	req *serviceports.SendEmailRequest,
) (*email.Message, error) {
	f.sent = req
	if f.err != nil {
		return nil, f.err
	}

	return &email.Message{}, nil
}

func (f *fakeMailer) SendPersisted(
	context.Context,
	*serviceports.SendPersistedEmailRequest,
) (*email.Message, error) {
	return nil, nil
}

type fakeRenderer struct {
	serviceports.DocumentTemplateResolver

	req *serviceports.RenderMessageRequest
}

func (f *fakeRenderer) RenderMessage(
	_ context.Context,
	req *serviceports.RenderMessageRequest,
) (*serviceports.RenderedMessage, error) {
	f.req = req

	return &serviceports.RenderedMessage{
		Subject: "[Acme] " + req.Data.(documenttemplate.AgentEmailContext).AgentSubject,
		HTML:    "<p>html</p>",
		Text:    "text",
	}, nil
}

type fakeCommentWriter struct {
	serviceports.ShipmentCommentService

	created *serviceports.CreateSystemShipmentCommentRequest
	// existing is what the repeat guard reads back before a send. Empty
	// means nothing has gone to this customer yet.
	existing []*shipment.ShipmentComment
}

func (f *fakeCommentWriter) ListByShipmentID(
	context.Context,
	*repositories.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return &pagination.CursorListResult[*shipment.ShipmentComment]{Items: f.existing}, nil
}

func (f *fakeCommentWriter) CreateSystem(
	_ context.Context,
	req *serviceports.CreateSystemShipmentCommentRequest,
) (*shipment.ShipmentComment, error) {
	f.created = req

	return &shipment.ShipmentComment{}, nil
}

func customerEmailFixture() (*emailCustomerTool, *fakeMailer, *fakeCustomerRepo, *fakeCommentWriter, *fakeRenderer) {
	customerID := pulid.MustNew("cus_")
	mailer := &fakeMailer{}
	customers := &fakeCustomerRepo{entity: &customer.Customer{
		ID:   customerID,
		Name: "Acme Freight",
		EmailProfile: &customer.CustomerEmailProfile{
			ToRecipients: "ops@acme.example, ap@acme.example",
		},
	}}
	comments := &fakeCommentWriter{}
	renderer := &fakeRenderer{}
	tool := &emailCustomerTool{deps: customerMailer{
		email:     mailer,
		templates: renderer,
		customers: customers,
		shipments: &fakeShipmentReader{entity: &shipment.Shipment{
			ID: pulid.MustNew("shp_"), ProNumber: "S12345", CustomerID: customerID,
		}},
		comments: comments,
	}}

	return tool, mailer, customers, comments, renderer
}

/*
The customer is written to at the addresses on the customer's own record, in
the organization's own letterhead, and the shipment's thread records that it
happened. The model supplies the words and nothing else about where they go.
*/
func TestEmailCustomer_WritesToTheCustomersContactsAndRecordsIt(t *testing.T) {
	t.Parallel()

	tool, mailer, customers, comments, renderer := customerEmailFixture()
	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 running about an hour late",
		"body":       "The truck is held in traffic; we now expect 3:30 PM.",
	})
	params.IdempotencyKey = "idem-1"

	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, mailer.sent)
	assert.Equal(t, []string{"ops@acme.example", "ap@acme.example"}, mailer.sent.To)
	assert.Equal(t, email.PurposeOperations, mailer.sent.Purpose)
	assert.Equal(t, "[Acme] S12345 running about an hour late", mailer.sent.Subject)
	assert.Equal(t, "idem-1", mailer.sent.IdempotencyKey)
	assert.True(t, customers.req.IncludeEmailProfile)

	require.NotNil(t, renderer.req)
	assert.Equal(t, documenttemplate.KindAgentCustomerUpdateEmail, renderer.req.Kind)
	context, ok := renderer.req.Data.(documenttemplate.AgentEmailContext)
	require.True(t, ok)
	assert.Equal(t, "Acme Freight", context.CustomerName)
	assert.Equal(t, "S12345", context.ShipmentProNumber)

	require.NotNil(t, comments.created)
	assert.Equal(t, shipment.CommentTypeCustomerUpdate, comments.created.Type)
	assert.Contains(t, comments.created.Comment, "ops@acme.example")
	assert.Contains(t, comments.created.Comment, "we now expect 3:30 PM")
}

func TestEmailCustomer_RefusesACustomerWithNoContacts(t *testing.T) {
	t.Parallel()

	tool, mailer, customers, _, _ := customerEmailFixture()
	customers.entity.EmailProfile = nil
	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "Update",
		"body":       "Body.",
	})
	params.IdempotencyKey = "idem-3"

	err := tool.Execute(t.Context(), params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no notice recipients")
	assert.Nil(t, mailer.sent)
}

func TestEmailCustomer_IsAProposalThatNeedsAnIdempotencyKey(t *testing.T) {
	t.Parallel()

	tool, mailer, _, _, _ := customerEmailFixture()
	assert.Equal(t, agent.TierPropose, tool.Policy().DefaultTier)
	assert.True(t, tool.Policy().Idempotent)
	assert.Equal(t, permission.ResourceCustomerCommunication, tool.Policy().Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)

	err := tool.Execute(t.Context(), executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "Update",
		"body":       "Body.",
	}))

	require.ErrorIs(t, err, ErrMissingIdempotencyKey)
	assert.Nil(t, mailer.sent)
}

func TestEmailCustomer_ReportsARecordFailureAfterTheSend(t *testing.T) {
	t.Parallel()

	tool, mailer, _, _, _ := customerEmailFixture()
	tool.deps.comments = &failingCommentWriter{}
	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "Update",
		"body":       "Body.",
	})
	params.IdempotencyKey = "idem-2"

	err := tool.Execute(t.Context(), params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "was sent but could not be recorded")
	assert.NotNil(t, mailer.sent, "the send is not undone by a failed record")
}

type failingCommentWriter struct {
	serviceports.ShipmentCommentService
}

// Nothing has gone to this customer yet; the failure under test is the
// record after the send, not the guard before it.
func (f *failingCommentWriter) ListByShipmentID(
	context.Context,
	*repositories.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return &pagination.CursorListResult[*shipment.ShipmentComment]{}, nil
}

func (failingCommentWriter) CreateSystem(
	context.Context,
	*serviceports.CreateSystemShipmentCommentRequest,
) (*shipment.ShipmentComment, error) {
	return nil, errors.New("thread locked")
}

type fakeDetention struct {
	noticed *detentionservice.SendOccurrenceNoticeParams
	waived  *detentionservice.WaiveParams
}

func (f *fakeDetention) SendOccurrenceNotice(
	_ context.Context,
	params detentionservice.SendOccurrenceNoticeParams,
) (*detention.DetentionOccurrence, error) {
	f.noticed = &params

	return &detention.DetentionOccurrence{}, nil
}

func (f *fakeDetention) Waive(
	_ context.Context,
	params detentionservice.WaiveParams,
) (*detention.DetentionOccurrence, error) {
	f.waived = &params

	return &detention.DetentionOccurrence{}, nil
}

func TestSendDetentionNotice_SendsAsTheActorNotTheSweep(t *testing.T) {
	t.Parallel()

	det := &fakeDetention{}
	occurrenceID := pulid.MustNew("dto_")
	params := executeParams(map[string]any{"occurrenceId": occurrenceID.String()})

	require.NoError(t, newSendDetentionNoticeTool(det).Execute(t.Context(), params))

	require.NotNil(t, det.noticed)
	assert.Equal(t, occurrenceID, det.noticed.OccurrenceID)
	assert.Equal(t, params.Actor.UserID, det.noticed.UserID)
	assert.False(t, det.noticed.Automatic)
}

func TestWaiveDetention_RequiresACodedReasonAndANote(t *testing.T) {
	t.Parallel()

	det := &fakeDetention{}
	tool := newWaiveDetentionTool(det)
	occurrenceID := pulid.MustNew("dto_")

	require.NoError(t, tool.Execute(t.Context(), approvedParams(map[string]any{
		"occurrenceId": occurrenceID.String(),
		"reason":       "CarrierFault",
		"note":         " Our truck arrived two hours late. ",
	})))
	require.NotNil(t, det.waived)
	assert.Equal(t, detention.WaiverReasonCarrierFault, det.waived.Reason)
	assert.Equal(t, "Our truck arrived two hours late.", det.waived.Note)

	det.waived = nil
	err := tool.Execute(t.Context(), approvedParams(map[string]any{
		"occurrenceId": occurrenceID.String(),
		"reason":       "BecauseISaidSo",
		"note":         "x",
	}))
	require.Error(t, err)
	assert.Nil(t, det.waived)
	assert.Equal(t, agent.TierPropose, tool.Policy().DefaultTier)
}

func TestWaiveDetention_GivesUpRevenueOnlyOnceAPersonApproves(t *testing.T) {
	t.Parallel()

	det := &fakeDetention{}
	tool := newWaiveDetentionTool(det)

	err := tool.Execute(t.Context(), executeParams(map[string]any{
		"occurrenceId": pulid.MustNew("dto_").String(),
		"reason":       "CarrierFault",
		"note":         "Our truck arrived two hours late.",
	}))
	require.ErrorIs(t, err, ErrApprovalNeedsAPerson)
	assert.Nil(t, det.waived)

	policy := tool.Policy()
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
}

func approvedParams(params map[string]any) serviceports.ToolExecuteParams {
	execute := executeParams(params)
	execute.ProposalID = pulid.MustNew("agp_")

	return execute
}

func TestMonitoringActionTools_RejectAMismatchedActor(t *testing.T) {
	t.Parallel()

	tool, _, _, _, _ := customerEmailFixture()
	for _, candidate := range []serviceports.AgentTool{
		newEvaluateServiceFailuresTool(&fakeFailureDecider{}),
		newResolveServiceFailureTool(&fakeFailureDecider{}),
		newNotifyDriverTool(&fakeDriverNotifier{}),
		tool,
		newSendDetentionNoticeTool(&fakeDetention{}),
		newWaiveDetentionTool(&fakeDetention{}),
	} {
		params := executeParams(map[string]any{})
		params.Actor.BusinessUnitID = pulid.MustNew("bu_")
		require.ErrorIs(
			t,
			candidate.Execute(t.Context(), params),
			ErrTenantMismatch,
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

	customerTool, _, _, _, _ := customerEmailFixture()
	tools := []struct {
		tool     serviceports.AgentTool
		resource permission.Resource
	}{
		{newNotifyDriverTool(&fakeDriverNotifier{}), permission.ResourceDriverMessage},
		{customerTool, permission.ResourceCustomerCommunication},
		{newSendDetentionNoticeTool(&fakeDetention{}), permission.ResourceCustomerCommunication},
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

	tool, mailer, _, comments, _ := customerEmailFixture()
	comments.existing = []*shipment.ShipmentComment{{
		CreatedAt: timeutils.NowUnix() - 600,
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "agent", "tool": "email_customer"},
	}}

	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 has departed",
		"body":       "The truck left the yard at 3:32 PM.",
	})
	params.IdempotencyKey = "idem-2"

	err := tool.Execute(t.Context(), params)
	require.ErrorIs(t, err, ErrCustomerAlreadyTold)
	assert.Nil(t, mailer.sent, "nothing may go out once the customer has been told")
	assert.Nil(t, comments.created)
}

// A dispatcher typing an update on the shipment is not an email to the
// customer, and must not silence one.
func TestEmailCustomer_ATypedCommentDoesNotSilenceTheDesk(t *testing.T) {
	t.Parallel()

	tool, mailer, _, comments, _ := customerEmailFixture()
	comments.existing = []*shipment.ShipmentComment{{
		CreatedAt: timeutils.NowUnix() - 60,
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "user"},
	}}

	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 has departed",
		"body":       "The truck left the yard at 3:32 PM.",
	})
	params.IdempotencyKey = "idem-3"

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, mailer.sent)
}
