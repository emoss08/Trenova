package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type FakeFailureDecider struct {
	Existing      *servicefailure.ServiceFailure                              `json:"-"`
	Evaluated     *servicefailureports.EvaluateShipmentServiceFailuresRequest `json:"-"`
	EvaluatedStop *servicefailureports.EvaluateStopServiceFailuresRequest     `json:"-"`
	Bulk          *servicefailureports.BulkEvaluateServiceFailuresRequest     `json:"-"`
	Resolved      *servicefailureports.ServiceFailureLifecycleRequest         `json:"-"`
	Actor         *permissionports.RequestActor                               `json:"-"`
	plans         map[pulid.ID]*servicefailureports.ServiceFailureDetectionPlan
}

func (f *FakeFailureDecider) EvaluateStop(
	_ context.Context,
	req *servicefailureports.EvaluateStopServiceFailuresRequest,
	actor *permissionports.RequestActor,
) (*servicefailureports.ServiceFailureEvaluationResult, error) {
	f.EvaluatedStop = req
	f.Actor = actor

	return &servicefailureports.ServiceFailureEvaluationResult{}, nil
}

func (f *FakeFailureDecider) BulkEvaluate(
	_ context.Context,
	req *servicefailureports.BulkEvaluateServiceFailuresRequest,
	actor *permissionports.RequestActor,
) (*servicefailureports.ServiceFailureEvaluationResult, error) {
	f.Bulk = req
	f.Actor = actor

	return &servicefailureports.ServiceFailureEvaluationResult{}, nil
}

func (f *FakeFailureDecider) EvaluateShipment(
	_ context.Context,
	req *servicefailureports.EvaluateShipmentServiceFailuresRequest,
	actor *permissionports.RequestActor,
) (*servicefailureports.ServiceFailureEvaluationResult, error) {
	f.Evaluated = req
	f.Actor = actor

	return &servicefailureports.ServiceFailureEvaluationResult{}, nil
}

func (f *FakeFailureDecider) GetByID(
	_ context.Context,
	req *servicefailureports.GetServiceFailureByIDRequest,
) (*servicefailure.ServiceFailure, error) {
	if f.Existing != nil && f.Existing.ID == req.ID {
		return f.Existing, nil
	}

	return nil, errortypes.NewNotFoundError("ServiceFailure not found")
}

func (f *FakeFailureDecider) Resolve(
	_ context.Context,
	req *servicefailureports.ServiceFailureLifecycleRequest,
	actor *permissionports.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	f.Resolved = req
	f.Actor = actor

	return f.Existing, nil
}

func (f *FakeFailureDecider) PreviewEvaluateShipment(
	_ context.Context,
	req *servicefailureports.EvaluateShipmentServiceFailuresRequest,
) (*servicefailureports.ServiceFailureDetectionPlan, error) {
	if plan, ok := f.plans[req.ShipmentID]; ok {
		return plan, nil
	}

	return &servicefailureports.ServiceFailureDetectionPlan{}, nil
}

type fakeCustomerRepo struct {
	customerports.CustomerRepository

	Entity *customer.Customer                   `json:"-"`
	Req    customerports.GetCustomerByIDRequest `json:"-"`
}

func (f *fakeCustomerRepo) GetByID(
	_ context.Context,
	req customerports.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	f.Req = req

	return f.Entity, nil
}

type fakeShipmentReader struct {
	shipmentports.ShipmentRepository

	entity *shipment.Shipment
}

func (f *fakeShipmentReader) GetByID(
	_ context.Context,
	_ *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return f.entity, nil
}

type FakeMailer struct {
	Sent *emailports.SendEmailRequest `json:"-"`
	Err  error                        `json:"-"`
}

func (f *FakeMailer) Send(
	_ context.Context,
	req *emailports.SendEmailRequest,
) (*email.Message, error) {
	f.Sent = req
	if f.Err != nil {
		return nil, f.Err
	}

	return &email.Message{}, nil
}

func (f *FakeMailer) SendPersisted(
	context.Context,
	*emailports.SendPersistedEmailRequest,
) (*email.Message, error) {
	return nil, nil
}

type FakeRenderer struct {
	documenttemplateports.DocumentTemplateResolver

	Req *documenttemplateports.RenderMessageRequest `json:"-"`
}

func (f *FakeRenderer) RenderMessage(
	_ context.Context,
	req *documenttemplateports.RenderMessageRequest,
) (*documenttemplateports.RenderedMessage, error) {
	f.Req = req

	return &documenttemplateports.RenderedMessage{
		Subject: "[Acme] " + req.Data.(documenttemplate.AgentEmailContext).AgentSubject,
		HTML:    "<p>html</p>",
		Text:    "text",
	}, nil
}

type fakeCommentWriter struct {
	shipmentports.ShipmentCommentService

	Created *shipmentports.CreateSystemShipmentCommentRequest `json:"-"`
	// existing is what the repeat guard reads back before a send. Empty
	// means nothing has gone to this customer yet.
	Existing []*shipment.ShipmentComment `json:"-"`
}

func (f *fakeCommentWriter) ListByShipmentID(
	context.Context,
	*shipmentports.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return &pagination.CursorListResult[*shipment.ShipmentComment]{Items: f.Existing}, nil
}

func (f *fakeCommentWriter) CreateSystem(
	_ context.Context,
	req *shipmentports.CreateSystemShipmentCommentRequest,
) (*shipment.ShipmentComment, error) {
	f.Created = req

	return &shipment.ShipmentComment{}, nil
}

func CustomerEmailFixture() (*operationstools.EmailCustomerTool, *FakeMailer, *fakeCustomerRepo, *fakeCommentWriter, *FakeRenderer) {
	customerID := pulid.MustNew("cus_")
	mailer := &FakeMailer{}
	customers := &fakeCustomerRepo{Entity: &customer.Customer{
		ID:   customerID,
		Name: "Acme Freight",
		EmailProfile: &customer.CustomerEmailProfile{
			ToRecipients: "ops@acme.example, ap@acme.example",
		},
	}}
	comments := &fakeCommentWriter{}
	renderer := &FakeRenderer{}
	tool := &operationstools.EmailCustomerTool{Deps: operationstools.CustomerMailer{
		Email:     mailer,
		Templates: renderer,
		Customers: customers,
		Shipments: &fakeShipmentReader{entity: &shipment.Shipment{
			ID: pulid.MustNew("shp_"), ProNumber: "S12345", CustomerID: customerID,
		}},
		Comments: comments,
	}}

	return tool, mailer, customers, comments, renderer
}

type FakeDetention struct {
	Noticed  *detentionservice.SendOccurrenceNoticeParams `json:"-"`
	Waived   *detentionservice.WaiveParams                `json:"-"`
	Disputed *detentionservice.DisputeParams              `json:"-"`
	Guard    *WriteGuard                                  `json:"-"`

	PreviewedNotice *detentionservice.SendOccurrenceNoticeParams `json:"-"`
	PreviewedWaive  *detentionservice.WaiveParams                `json:"-"`
	Notice          *detentionservice.NoticePreview              `json:"-"`
	Change          *detentionservice.OccurrenceChange           `json:"-"`
}

func (f *FakeDetention) SendOccurrenceNotice(
	_ context.Context,
	params detentionservice.SendOccurrenceNoticeParams,
) (*detention.DetentionOccurrence, error) {
	f.Noticed = &params

	return &detention.DetentionOccurrence{}, nil
}

func (f *FakeDetention) Waive(
	_ context.Context,
	params detentionservice.WaiveParams,
) (*detention.DetentionOccurrence, error) {
	f.Waived = &params

	return &detention.DetentionOccurrence{}, nil
}

func (f *FakeDetention) PreviewOccurrenceNotice(
	_ context.Context,
	params *detentionservice.SendOccurrenceNoticeParams,
) (*detentionservice.NoticePreview, error) {
	f.PreviewedNotice = params

	return f.Notice, nil
}

func (f *FakeDetention) Dispute(
	_ context.Context,
	params detentionservice.DisputeParams,
) (*detention.DetentionOccurrence, error) {
	if err := f.Guard.Write(); err != nil {
		return nil, err
	}
	f.Disputed = &params

	return &detention.DetentionOccurrence{ID: params.OccurrenceID}, nil
}

func (f *FakeDetention) PreviewDispute(
	_ context.Context,
	params *detentionservice.DisputeParams,
) (*detentionservice.OccurrenceChange, error) {
	if f.Change != nil {
		return f.Change, nil
	}
	before := &detention.DetentionOccurrence{
		ID:                params.OccurrenceID,
		ShipmentProNumber: "PRO-3001",
		LocationName:      "Acme DC",
		BillableAmount:    decimal.RequireFromString("150"),
		Currency:          "USD",
		Status:            detention.OccurrenceStatusPending,
		Version:           2,
	}
	after := *before
	if err := after.Dispute(params.Note, 1_700_000_000); err != nil {
		return nil, err
	}

	return &detentionservice.OccurrenceChange{Before: before, After: &after, Now: 1_700_000_000}, nil
}

func (f *FakeDetention) PreviewWaive(
	_ context.Context,
	params *detentionservice.WaiveParams,
) (*detentionservice.OccurrenceChange, error) {
	f.PreviewedWaive = params

	return f.Change, nil
}

func ApprovedParams(params map[string]any) agentports.ToolExecuteParams {
	execute := ExecuteParams(params)
	execute.ProposalID = pulid.MustNew("agp_")

	return execute
}
