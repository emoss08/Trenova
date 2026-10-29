package customertrackingservice

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const testNow = int64(1_800_000_000)

var testTenant = pagination.TenantInfo{
	OrgID:  pulid.MustNew("org_"),
	BuID:   pulid.MustNew("bu_"),
	UserID: pulid.MustNew("usr_"),
}

type memoryProfiles struct {
	customertrackingports.ProfileRepository
	rows map[pulid.ID]*customertracking.Profile
}

func (r *memoryProfiles) GetByCustomer(
	_ context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
) (*customertracking.Profile, error) {
	if row := r.rows[customerID]; row != nil {
		copied := *row
		return &copied, nil
	}
	return nil, nil
}

func (r *memoryProfiles) Create(_ context.Context, entity *customertracking.Profile) error {
	copied := *entity
	r.rows[entity.CustomerID] = &copied
	return nil
}

func (r *memoryProfiles) Update(_ context.Context, entity *customertracking.Profile) error {
	current := r.rows[entity.CustomerID]
	if current == nil || current.Version != entity.Version {
		return errortypes.NewBusinessError("version mismatch")
	}
	entity.Version++
	copied := *entity
	r.rows[entity.CustomerID] = &copied
	return nil
}

type memoryLinks struct {
	customertrackingports.LinkRepository
	rows  []*customertracking.Link
	views int
}

func (r *memoryLinks) Create(_ context.Context, entity *customertracking.Link) error {
	copied := *entity
	r.rows = append(r.rows, &copied)
	return nil
}

func (r *memoryLinks) GetByHash(_ context.Context, hash string) (*customertracking.Link, error) {
	for _, row := range r.rows {
		if row.TokenHash == hash {
			copied := *row
			return &copied, nil
		}
	}
	return nil, nil
}

func (r *memoryLinks) Revoke(
	_ context.Context,
	req *customertrackingports.RevokeLinksRequest,
) (int, error) {
	count := 0
	for _, row := range r.rows {
		if row.ShipmentID != req.ShipmentID || row.RevokedAt != nil {
			continue
		}
		if !req.LinkID.IsNil() && row.ID != req.LinkID {
			continue
		}
		at := req.At
		row.RevokedAt = &at
		count++
	}
	return count, nil
}

func (r *memoryLinks) RecordView(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	_ int64,
) error {
	r.views++
	return nil
}

type memoryNotifications struct {
	customertrackingports.NotificationRepository
	rows []*customertracking.Notification
}

func (r *memoryNotifications) InsertOnce(
	_ context.Context,
	entity *customertracking.Notification,
) (bool, error) {
	for _, row := range r.rows {
		if row.ShipmentID == entity.ShipmentID && row.Event == entity.Event &&
			row.SubjectKey == entity.SubjectKey {
			return false, nil
		}
	}
	copied := *entity
	r.rows = append(r.rows, &copied)
	return true, nil
}

func (r *memoryNotifications) ListDue(
	_ context.Context,
	_ pagination.TenantInfo,
	now int64,
	_ int,
) ([]*customertracking.Notification, error) {
	due := make([]*customertracking.Notification, 0)
	for _, row := range r.rows {
		if row.Status == customertracking.NotificationStatusPending && row.NextAttemptAt <= now {
			copied := *row
			due = append(due, &copied)
		}
	}
	return due, nil
}

func (r *memoryNotifications) Record(
	_ context.Context,
	outcome *customertrackingports.DeliveryOutcome,
) (bool, error) {
	for _, row := range r.rows {
		if row.ID != outcome.ID {
			continue
		}
		if row.Status != customertracking.NotificationStatusPending {
			return false, nil
		}
		row.Status = outcome.Status
		row.SkipReason = outcome.SkipReason
		row.Recipients = outcome.Recipients
		row.LinkID = outcome.LinkID
		row.EmailMessageID = outcome.EmailMessageID
		row.Attempts = outcome.Attempts
		row.LastError = outcome.LastError
		row.NextAttemptAt = outcome.NextAttemptAt
		row.SentAt = outcome.SentAt
		return true, nil
	}
	return false, nil
}

func (r *memoryNotifications) byEvent(event customertracking.Event) *customertracking.Notification {
	for _, row := range r.rows {
		if row.Event == event {
			return row
		}
	}
	return nil
}

type fakeFacts struct {
	customertrackingports.FactRepository
	pods        map[pulid.ID]*customertrackingports.PodDocument
	holds       []customertrackingports.PublicHold
	notes       []customertrackingports.PublicNote
	location    *customertrackingports.PublicLocation
	descriptors customertrackingports.ShipmentDescriptors
}

func (f *fakeFacts) LatestPods(
	_ context.Context,
	_ pagination.TenantInfo,
	_ []pulid.ID,
) (map[pulid.ID]*customertrackingports.PodDocument, error) {
	if f.pods == nil {
		return map[pulid.ID]*customertrackingports.PodDocument{}, nil
	}
	return f.pods, nil
}

func (f *fakeFacts) CustomerVisibleHolds(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) ([]customertrackingports.PublicHold, error) {
	return f.holds, nil
}

func (f *fakeFacts) CustomerVisibleNotes(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	_ int,
) ([]customertrackingports.PublicNote, error) {
	return f.notes, nil
}

func (f *fakeFacts) LastReportedLocation(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*customertrackingports.PublicLocation, error) {
	return f.location, nil
}

func (f *fakeFacts) Descriptors(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*customertrackingports.ShipmentDescriptors, error) {
	copied := f.descriptors
	return &copied, nil
}

type fakeShipments struct {
	shipmentports.ShipmentRepository
	rows map[pulid.ID]*shipment.Shipment
}

func (r *fakeShipments) GetByID(
	_ context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	row := r.rows[req.ID]
	if row == nil {
		return nil, errortypes.NewNotFoundError("Shipment not found")
	}
	return row, nil
}

type memoryStopEtas struct {
	stopetaports.Repository
	rows []*stopeta.StopETA
}

func (f *memoryStopEtas) ListByShipmentIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	_ []pulid.ID,
) ([]*stopeta.StopETA, error) {
	return f.rows, nil
}

type fakeCustomers struct {
	customerports.CustomerRepository
	rows map[pulid.ID]*customer.Customer
}

func (r *fakeCustomers) GetByID(
	_ context.Context,
	req customerports.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	row := r.rows[req.ID]
	if row == nil {
		return nil, errortypes.NewNotFoundError("Customer not found")
	}
	return row, nil
}

type renderedCall struct {
	kind string
	data any
}

type fakeTemplates struct {
	documenttemplateports.DocumentTemplateResolver
	calls []renderedCall
}

func (f *fakeTemplates) RenderMessage(
	_ context.Context,
	req *documenttemplateports.RenderMessageRequest,
) (*documenttemplateports.RenderedMessage, error) {
	f.calls = append(f.calls, renderedCall{kind: string(req.Kind), data: req.Data})
	return &documenttemplateports.RenderedMessage{
		Subject: "subject",
		HTML:    "<p>body</p>",
		Text:    "body",
	}, nil
}

// fakeEmail mimics the email service's idempotency: a repeated key replays the
// first message instead of sending another.
type fakeEmail struct {
	emailports.EmailService
	sent     []*emailports.SendEmailRequest
	byKey    map[string]*email.Message
	failures int
}

func (f *fakeEmail) Send(
	_ context.Context,
	req *emailports.SendEmailRequest,
) (*email.Message, error) {
	if f.failures > 0 {
		f.failures--
		return nil, errors.New("provider unavailable")
	}
	if f.byKey == nil {
		f.byKey = map[string]*email.Message{}
	}
	if message, ok := f.byKey[req.IdempotencyKey]; ok {
		return message, nil
	}
	message := &email.Message{ID: pulid.MustNew("emsg_")}
	f.byKey[req.IdempotencyKey] = message
	f.sent = append(f.sent, req)
	return message, nil
}

type fakeDocuments struct {
	documentports.InvoiceDocumentService
	read []pulid.ID
}

func (f *fakeDocuments) GetDownloadContent(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	f.read = append(f.read, req.ID)
	return &documentports.DocumentContent{
		Document:           &document.Document{ID: req.ID, OriginalName: "pod.pdf"},
		Body:               io.NopCloser(strings.NewReader("pdf")),
		ContentType:        "application/pdf",
		ContentLength:      3,
		ContentDisposition: `attachment; filename="pod.pdf"`,
	}, nil
}

type harness struct {
	service       *Service
	profiles      *memoryProfiles
	links         *memoryLinks
	notifications *memoryNotifications
	facts         *fakeFacts
	shipments     *fakeShipments
	etas          *memoryStopEtas
	customers     *fakeCustomers
	templates     *fakeTemplates
	email         *fakeEmail
	documents     *fakeDocuments
	now           int64
}

func newHarness() *harness {
	h := &harness{
		profiles:      &memoryProfiles{rows: map[pulid.ID]*customertracking.Profile{}},
		links:         &memoryLinks{},
		notifications: &memoryNotifications{},
		facts:         &fakeFacts{},
		shipments:     &fakeShipments{rows: map[pulid.ID]*shipment.Shipment{}},
		etas:          &memoryStopEtas{},
		customers:     &fakeCustomers{rows: map[pulid.ID]*customer.Customer{}},
		templates:     &fakeTemplates{},
		email:         &fakeEmail{},
		documents:     &fakeDocuments{},
		now:           testNow,
	}
	cfg := &config.Config{}
	cfg.App.WebBaseURL = "https://app.example.com/"
	h.service = &Service{
		l:               zap.NewNop(),
		appConfig:       &cfg.App,
		tenderingConfig: &cfg.Tendering,
		profiles:        h.profiles,
		links:           h.links,
		notifications:   h.notifications,
		facts:           h.facts,
		shipmentRepo:    h.shipments,
		stopEtas:        h.etas,
		customerRepo:    h.customers,
		templates:       h.templates,
		emailService:    h.email,
		documents:       h.documents,
		now:             func() int64 { return h.now },
	}
	return h
}

func at(offset int64) *int64 {
	v := testNow + offset
	return &v
}

type stopSpec struct {
	stopType  shipmentvocab.StopType
	city      string
	state     string
	arrival   *int64
	departure *int64
}

// addShipment stores a one-move shipment for a new customer and returns both.
func (h *harness) addShipment(status shipmentvocab.Status, stops ...stopSpec) *shipment.Shipment {
	customerID := pulid.MustNew("cus_")
	h.customers.rows[customerID] = &customer.Customer{
		ID:   customerID,
		Name: "Halstead Grocery",
	}
	move := &shipment.ShipmentMove{ID: pulid.MustNew("smv_"), Sequence: 1}
	for i, spec := range stops {
		move.Stops = append(move.Stops, &shipment.Stop{
			ID:                   pulid.MustNew("stp_"),
			Type:                 spec.stopType,
			Sequence:             int64(i + 1),
			ScheduledWindowStart: testNow + int64(i)*3600,
			ActualArrival:        spec.arrival,
			ActualDeparture:      spec.departure,
			Location: &location.Location{
				Name:     spec.city + " DC",
				City:     spec.city,
				Timezone: "America/Chicago",
				State:    &usstate.UsState{Abbreviation: spec.state},
			},
		})
	}
	sh := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: testTenant.OrgID,
		BusinessUnitID: testTenant.BuID,
		CustomerID:     customerID,
		Status:         status,
		ProNumber:      "PRO-77",
		BOL:            "BOL-1",
		Moves:          []*shipment.ShipmentMove{move},
		Customer:       h.customers.rows[customerID],
	}
	h.shipments.rows[sh.ID] = sh
	return sh
}

func (h *harness) notify(customerID pulid.ID, configure func(*customertracking.Profile)) {
	enabled := testNow - 24*3600
	profile := customertracking.DefaultProfile(testTenant.OrgID, testTenant.BuID, customerID)
	profile.ID = customertracking.NewProfileID()
	profile.Recipients = "ops@halstead.example"
	profile.NotificationsEnabledAt = &enabled
	configure(profile)
	h.profiles.rows[customerID] = profile
}
