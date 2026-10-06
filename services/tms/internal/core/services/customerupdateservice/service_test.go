package customerupdateservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testNow = int64(1_767_225_600)

type shipmentReader struct {
	repositories.ShipmentRepository
	entity *shipment.Shipment
}

func (f *shipmentReader) GetByID(
	context.Context,
	*repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return f.entity, nil
}

type customerReader struct {
	repositories.CustomerRepository
	entity *customer.Customer
}

func (f *customerReader) GetByID(
	context.Context,
	repositories.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	return f.entity, nil
}

type renderer struct {
	services.DocumentTemplateResolver
	data documenttemplate.AgentEmailContext
}

func (f *renderer) RenderMessage(
	_ context.Context,
	req *services.RenderMessageRequest,
) (*services.RenderedMessage, error) {
	f.data = req.Data.(documenttemplate.AgentEmailContext)

	return &services.RenderedMessage{
		Subject: f.data.AgentSubject,
		HTML:    "<p>" + f.data.AgentBody + "</p>",
		Text:    f.data.AgentBody,
	}, nil
}

type mailer struct {
	services.EmailService
	sent []*services.SendEmailRequest
	err  error
}

func (f *mailer) Send(_ context.Context, req *services.SendEmailRequest) (*email.Message, error) {
	f.sent = append(f.sent, req)
	if f.err != nil {
		return nil, f.err
	}

	return &email.Message{}, nil
}

type commentBook struct {
	services.ShipmentCommentService
	existing []*shipment.ShipmentComment
	created  []*services.CreateSystemShipmentCommentRequest
}

func (f *commentBook) ListByShipmentID(
	context.Context,
	*repositories.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return &pagination.CursorListResult[*shipment.ShipmentComment]{Items: f.existing}, nil
}

func (f *commentBook) CreateSystem(
	_ context.Context,
	req *services.CreateSystemShipmentCommentRequest,
) (*shipment.ShipmentComment, error) {
	f.created = append(f.created, req)

	return &shipment.ShipmentComment{}, nil
}

type fixture struct {
	service  *Service
	mailer   *mailer
	comments *commentBook
	renderer *renderer
	request  *NotifyDelayRequest
	actor    *services.RequestActor
}

func newFixture(recipients string) *fixture {
	customerID := pulid.MustNew("cus_")
	shipmentID := pulid.MustNew("shp_")
	f := &fixture{
		mailer:   &mailer{},
		comments: &commentBook{},
		renderer: &renderer{},
		actor:    &services.RequestActor{UserID: pulid.MustNew("usr_")},
	}
	f.service = &Service{
		email:     f.mailer,
		templates: f.renderer,
		customers: &customerReader{entity: &customer.Customer{
			ID:           customerID,
			Name:         "Cargill Protein",
			EmailProfile: &customer.CustomerEmailProfile{ToRecipients: recipients},
		}},
		shipments: &shipmentReader{entity: &shipment.Shipment{
			ID: shipmentID, ProNumber: "S2610-0412", CustomerID: customerID,
		}},
		comments: f.comments,
		now:      func() int64 { return testNow },
	}
	f.request = &NotifyDelayRequest{
		TenantInfo:     pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		ShipmentID:     shipmentID,
		Message:        "  The load is running about three hours behind the storm over I-80.  ",
		IdempotencyKey: "idem-1",
	}

	return f
}

func TestNotifyDelay_EmailsTheCustomersContactsAndRecordsIt(t *testing.T) {
	t.Parallel()

	f := newFixture("ops@cargill.example, dock@cargill.example")
	require.NoError(t, f.service.NotifyDelay(t.Context(), f.request, f.actor))

	require.Len(t, f.mailer.sent, 1)
	sent := f.mailer.sent[0]
	assert.Equal(t, []string{"ops@cargill.example", "dock@cargill.example"}, sent.To)
	assert.Equal(t, email.PurposeOperations, sent.Purpose)
	assert.True(t, sent.ProfileID.IsNil(), "the organization's operations profile sends it")
	assert.Equal(t, "idem-1", sent.IdempotencyKey)
	assert.Equal(t, "Delivery update for shipment S2610-0412", sent.Subject)
	assert.Equal(t,
		"The load is running about three hours behind the storm over I-80.",
		f.renderer.data.AgentBody,
	)

	require.Len(t, f.comments.created, 1)
	comment := f.comments.created[0]
	assert.Equal(t, shipment.CommentTypeCustomerUpdate, comment.Type)
	assert.Equal(t, shipment.CommentOriginBoard, comment.Metadata[shipment.CommentMetadataOrigin])
	assert.Equal(t, SourceDelayNotice, comment.Metadata[metadataSource])
	assert.Equal(t, f.actor.UserID.String(), comment.Metadata[metadataSentBy])
	assert.True(t, strings.HasPrefix(comment.Comment, "Emailed ops@cargill.example, dock@cargill.example:"))
}

func TestNotifyDelay_SendsNothingWhenTheCustomerWasJustTold(t *testing.T) {
	t.Parallel()

	f := newFixture("ops@cargill.example")
	f.comments.existing = []*shipment.ShipmentComment{{
		CreatedAt: testNow - 300,
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "agent", "tool": SourceAgentEmail},
	}}

	err := f.service.NotifyDelay(t.Context(), f.request, f.actor)

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Empty(t, f.mailer.sent)
	assert.Empty(t, f.comments.created)
}

func TestNotifyDelay_RefusesACustomerWithNoNoticeContacts(t *testing.T) {
	t.Parallel()

	f := newFixture("")
	err := f.service.NotifyDelay(t.Context(), f.request, f.actor)

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, err.Error(), "Cargill Protein has no notice recipients")
	assert.Empty(t, f.mailer.sent)
}

func TestNotifyDelay_ValidatesTheMessage(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"blank":    "   ",
		"too long": strings.Repeat("a", MaxMessageLength+1),
	}
	for name, message := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newFixture("ops@cargill.example")
			f.request.Message = message
			err := f.service.NotifyDelay(t.Context(), f.request, f.actor)

			var multi *errortypes.MultiError
			require.ErrorAs(t, err, &multi)
			assert.Empty(t, f.mailer.sent)
		})
	}
}

func TestNotifyDelay_RecordsNothingWhenTheSendFails(t *testing.T) {
	t.Parallel()

	f := newFixture("ops@cargill.example")
	f.mailer.err = errors.New("provider down")

	require.ErrorContains(t, f.service.NotifyDelay(t.Context(), f.request, f.actor), "provider down")
	assert.Empty(t, f.comments.created)
}
