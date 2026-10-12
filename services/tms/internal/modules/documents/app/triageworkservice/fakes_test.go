package triageworkservice

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var (
	testOrgID  = pulid.MustNew("org_")
	testBUID   = pulid.MustNew("bu_")
	testUserID = pulid.MustNew("usr_")
	testTenant = pagination.TenantInfo{OrgID: testOrgID, BuID: testBUID, UserID: testUserID}
)

type fakeMessages struct {
	inboundmessageports.InboundMessageRepository

	mu          sync.Mutex
	messages    map[pulid.ID]*inboundmessage.InboundMessage
	attachments map[pulid.ID]*inboundmessage.InboundAttachment
}

func (f *fakeMessages) attachmentsOf(messageID pulid.ID) []*inboundmessage.InboundAttachment {
	out := make([]*inboundmessage.InboundAttachment, 0, len(f.attachments))
	for _, attachment := range f.attachments {
		if attachment.MessageID == messageID {
			copied := *attachment
			out = append(out, &copied)
		}
	}
	slices.SortFunc(out, func(a, b *inboundmessage.InboundAttachment) int {
		return strings.Compare(a.FileName, b.FileName)
	})

	return out
}

func (f *fakeMessages) GetByID(
	_ context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	message, ok := f.messages[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("InboundMessage not found")
	}
	copied := *message
	if req.IncludeAttachments {
		copied.Attachments = f.attachmentsOf(message.ID)
	}

	return &copied, nil
}

func (f *fakeMessages) Update(
	_ context.Context,
	entity *inboundmessage.InboundMessage,
) (*inboundmessage.InboundMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entity.Version++
	stored := *entity
	stored.Attachments = nil
	f.messages[entity.ID] = &stored

	return entity, nil
}

func (f *fakeMessages) ListAttachments(
	_ context.Context,
	messageID pulid.ID,
	_ pagination.TenantInfo,
) ([]*inboundmessage.InboundAttachment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.attachmentsOf(messageID), nil
}

func (f *fakeMessages) GetAttachmentByID(
	_ context.Context,
	req inboundmessageports.GetInboundAttachmentRequest,
) (*inboundmessage.InboundAttachment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	attachment, ok := f.attachments[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Attachment not found")
	}
	copied := *attachment

	return &copied, nil
}

func (f *fakeMessages) UpdateAttachment(
	_ context.Context,
	entity *inboundmessage.InboundAttachment,
) (*inboundmessage.InboundAttachment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored := *entity
	f.attachments[entity.ID] = &stored

	return entity, nil
}

type fakeMailboxes struct {
	inboundmessageports.InboundMailboxRepository

	mailboxes map[pulid.ID]*inboundmessage.Mailbox
}

func (f *fakeMailboxes) GetByID(
	_ context.Context,
	req inboundmessageports.GetMailboxByIDRequest,
) (*inboundmessage.Mailbox, error) {
	mailbox, ok := f.mailboxes[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Mailbox not found")
	}
	copied := *mailbox
	copied.BlockedSenders = slices.Clone(mailbox.BlockedSenders)

	return &copied, nil
}

func (f *fakeMailboxes) Update(_ context.Context, entity *inboundmessage.Mailbox) (*inboundmessage.Mailbox, error) {
	stored := *entity
	f.mailboxes[entity.ID] = &stored

	return entity, nil
}

type fakeShipments struct {
	inboundmessageports.InboundShipmentFinder

	known map[pulid.ID]bool
}

func (f *fakeShipments) ShipmentExists(_ context.Context, _ pagination.TenantInfo, id pulid.ID) (bool, error) {
	return f.known[id], nil
}

type lineageCall struct {
	documentID   pulid.ID
	resourceType string
	resourceID   string
}

type fakeDocuments struct {
	mu      sync.Mutex
	repo    *fakeDocumentRepo
	content []byte
	lineage []lineageCall
	deleted []pulid.ID
}

func (f *fakeDocuments) AttachLineageToResource(
	_ context.Context,
	documentID pulid.ID,
	resourceType string,
	resourceID string,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*document.Document, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lineage = append(f.lineage, lineageCall{
		documentID:   documentID,
		resourceType: resourceType,
		resourceID:   resourceID,
	})
	doc := f.repo.docs[documentID]
	if doc == nil {
		doc = &document.Document{ID: documentID, Status: document.StatusActive}
		f.repo.docs[documentID] = doc
	}
	doc.ResourceType = resourceType
	doc.ResourceID = resourceID
	copied := *doc

	return &copied, nil
}

func (f *fakeDocuments) GetDownloadContent(
	context.Context,
	documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	return &documentports.DocumentContent{Body: io.NopCloser(bytes.NewReader(f.content))}, nil
}

func (f *fakeDocuments) Delete(_ context.Context, req documentports.DeleteDocumentRequest, _ pulid.ID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, req.ID)

	return nil
}

type fakeDocumentRepo struct {
	documentports.DocumentRepository

	docs map[pulid.ID]*document.Document
}

func (f *fakeDocumentRepo) GetByID(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	doc, ok := f.docs[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Document not found")
	}
	copied := *doc

	return &copied, nil
}

func (f *fakeDocumentRepo) Update(_ context.Context, entity *document.Document) (*document.Document, error) {
	stored := *entity
	f.docs[entity.ID] = &stored

	return entity, nil
}

type fakeDocumentTypes struct {
	documenttypeports.DocumentTypeRepository

	types map[pulid.ID]*documenttype.DocumentType
}

func (f *fakeDocumentTypes) GetByID(
	_ context.Context,
	req documenttypeports.GetDocumentTypeByIDRequest,
) (*documenttype.DocumentType, error) {
	docType, ok := f.types[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Document type not found")
	}

	return docType, nil
}

func (f *fakeDocumentTypes) GetByCode(
	_ context.Context,
	req documenttypeports.GetDocumentTypeByCodeRequest,
) (*documenttype.DocumentType, error) {
	for _, docType := range f.types {
		if docType.Code == req.Code {
			return docType, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Document type not found")
}

type fakeWork struct {
	inboundmessageports.TriageWorkRepository

	current []pulid.ID
}

func (f *fakeWork) CurrentShipmentDocuments(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	pulid.ID,
) ([]pulid.ID, error) {
	return f.current, nil
}

type fakePDF struct {
	pages     int
	assembled [][]captureports.CaptureAssemblyPage
}

func (f *fakePDF) PageCount(context.Context, []byte) (int, error) {
	return f.pages, nil
}

func (f *fakePDF) Split(context.Context, []byte) ([][]byte, error) {
	out := make([][]byte, 0, f.pages)
	for i := range f.pages {
		out = append(out, []byte{byte('a' + i)})
	}

	return out, nil
}

func (f *fakePDF) Assemble(_ context.Context, pages []captureports.CaptureAssemblyPage) ([]byte, error) {
	f.assembled = append(f.assembled, pages)
	out := make([]byte, 0, len(pages))
	for _, page := range pages {
		out = append(out, page.PDF...)
	}

	return out, nil
}

type fakeUploads struct {
	sessions map[pulid.ID]*documentuploadports.CreateSessionRequest
	parts    map[pulid.ID][]byte
}

func (f *fakeUploads) CreateSession(
	_ context.Context,
	req *documentuploadports.CreateSessionRequest,
) (*documentupload.DocumentUploadSession, error) {
	id := pulid.MustNew("dus_")
	f.sessions[id] = req

	return &documentupload.DocumentUploadSession{ID: id}, nil
}

func (f *fakeUploads) UploadPart(
	_ context.Context,
	req *documentuploadports.UploadPartRequest,
) (*documentupload.DocumentUploadSession, error) {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	f.parts[req.SessionID] = data

	return &documentupload.DocumentUploadSession{ID: req.SessionID}, nil
}

func (f *fakeUploads) Complete(
	_ context.Context,
	req *documentuploadports.CompletionRequest,
) (*documentupload.DocumentUploadSession, error) {
	documentID := pulid.MustNew("doc_")

	return &documentupload.DocumentUploadSession{ID: req.SessionID, DocumentID: &documentID}, nil
}

type fakeRecorder struct {
	mu         sync.Mutex
	activities []*triage.Activity
}

func (f *fakeRecorder) RecordActivity(_ context.Context, activities ...*triage.Activity) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.activities = append(f.activities, activities...)

	return nil
}

func (f *fakeRecorder) find(kind triage.ActivityKind) *triage.Activity {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, activity := range f.activities {
		if activity.Kind == kind {
			return activity
		}
	}

	return nil
}

type fakeNotifier struct {
	mu      sync.Mutex
	changed []pulid.ID
}

func (f *fakeNotifier) MessagesChanged(_ context.Context, messages ...*inboundmessage.InboundMessage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, message := range messages {
		f.changed = append(f.changed, message.ID)
	}
}

type fakeEvents struct {
	attached []pulid.ID
}

func (f *fakeEvents) ShipmentDelivered(context.Context, pagination.TenantInfo, pulid.ID) {}

func (f *fakeEvents) DocumentAttached(_ context.Context, _ pagination.TenantInfo, _, documentTypeID pulid.ID) {
	f.attached = append(f.attached, documentTypeID)
}

func (f *fakeEvents) InvoicePaid(context.Context, pagination.TenantInfo, pulid.ID) {}

type fakeDrafter struct {
	drafted []pulid.ID
}

func (f *fakeDrafter) DraftReplyFor(
	_ context.Context,
	message *inboundmessage.InboundMessage,
	_ triagenova.Mode,
) (*triage.Reply, error) {
	f.drafted = append(f.drafted, message.ID)

	return &triage.Reply{MessageID: message.ID}, nil
}

type fakeLoads struct {
	requests []*inboundmessageports.CreateTriageLoadRequest
	err      error
}

func (f *fakeLoads) CreateLoad(
	_ context.Context,
	req *inboundmessageports.CreateTriageLoadRequest,
) (*inboundmessageports.CreatedTriageLoad, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}

	return &inboundmessageports.CreatedTriageLoad{ShipmentID: pulid.MustNew("shp_"), ProNumber: "S-200001"}, nil
}

type fakeExceptions struct {
	requests []*inboundmessageports.LogTriageExceptionRequest
}

func (f *fakeExceptions) LogException(_ context.Context, req *inboundmessageports.LogTriageExceptionRequest) error {
	f.requests = append(f.requests, req)

	return nil
}

type fakeDetention struct {
	requests []*inboundmessageports.SettleTriageDetentionRequest
}

func (f *fakeDetention) SettleDetention(
	_ context.Context,
	req *inboundmessageports.SettleTriageDetentionRequest,
) (*inboundmessageports.SettledTriageDetention, error) {
	f.requests = append(f.requests, req)

	return &inboundmessageports.SettledTriageDetention{Amount: "150.00", Currency: "USD", Settled: true}, nil
}

type fixture struct {
	service    *Service
	messages   *fakeMessages
	mailboxes  *fakeMailboxes
	documents  *fakeDocuments
	docRepo    *fakeDocumentRepo
	work       *fakeWork
	pdf        *fakePDF
	uploads    *fakeUploads
	recorder   *fakeRecorder
	notifier   *fakeNotifier
	events     *fakeEvents
	drafter    *fakeDrafter
	loads      *fakeLoads
	exceptions *fakeExceptions
	detention  *fakeDetention
	message    *inboundmessage.InboundMessage
	attachment *inboundmessage.InboundAttachment
	shipmentID pulid.ID
	pod        *documenttype.DocumentType
	bol        *documenttype.DocumentType
}

func newFixture() *fixture {
	mailbox := &inboundmessage.Mailbox{
		ID:             pulid.MustNew("imbx_"),
		OrganizationID: testOrgID,
		BusinessUnitID: testBUID,
		Address:        "dispatch@trenova.example",
	}
	shipmentID := pulid.MustNew("shp_")
	message := &inboundmessage.InboundMessage{
		ID:                pulid.MustNew("imsg_"),
		OrganizationID:    testOrgID,
		BusinessUnitID:    testBUID,
		MailboxID:         mailbox.ID,
		FromAddress:       "Billing@Carrier.example",
		Subject:           "POD for S-104251",
		Classification:    inboundmessage.ClassificationProofOfDelivery,
		Status:            inboundmessage.StatusInReview,
		NovaRead:          true,
		MatchedShipmentID: shipmentID,
	}
	attachment := &inboundmessage.InboundAttachment{
		ID:             pulid.MustNew("iatt_"),
		OrganizationID: testOrgID,
		BusinessUnitID: testBUID,
		MessageID:      message.ID,
		DocumentID:     pulid.MustNew("doc_"),
		FileName:       "paperwork.pdf",
		ContentType:    "application/pdf",
		Kind:           inboundmessage.AttachmentProofOfDelivery,
	}
	pod := &documenttype.DocumentType{ID: pulid.MustNew("dt_"), Code: "POD", Name: "Proof of Delivery"}
	bol := &documenttype.DocumentType{ID: pulid.MustNew("dt_"), Code: "BOL", Name: "Bill of Lading"}

	docRepo := &fakeDocumentRepo{docs: map[pulid.ID]*document.Document{
		attachment.DocumentID: {ID: attachment.DocumentID, Status: document.StatusActive},
	}}
	f := &fixture{
		messages: &fakeMessages{
			messages:    map[pulid.ID]*inboundmessage.InboundMessage{message.ID: message},
			attachments: map[pulid.ID]*inboundmessage.InboundAttachment{attachment.ID: attachment},
		},
		mailboxes:  &fakeMailboxes{mailboxes: map[pulid.ID]*inboundmessage.Mailbox{mailbox.ID: mailbox}},
		docRepo:    docRepo,
		documents:  &fakeDocuments{repo: docRepo, content: []byte("%PDF")},
		work:       &fakeWork{},
		pdf:        &fakePDF{pages: 4},
		uploads:    &fakeUploads{sessions: map[pulid.ID]*documentuploadports.CreateSessionRequest{}, parts: map[pulid.ID][]byte{}},
		recorder:   &fakeRecorder{},
		notifier:   &fakeNotifier{},
		events:     &fakeEvents{},
		drafter:    &fakeDrafter{},
		loads:      &fakeLoads{},
		exceptions: &fakeExceptions{},
		detention:  &fakeDetention{},
		message:    message,
		attachment: attachment,
		shipmentID: shipmentID,
		pod:        pod,
		bol:        bol,
	}
	f.service = &Service{
		l:             zap.NewNop(),
		messages:      f.messages,
		mailboxes:     f.mailboxes,
		work:          f.work,
		activities:    f.recorder,
		notifier:      f.notifier,
		shipments:     &fakeShipments{known: map[pulid.ID]bool{shipmentID: true}},
		documentRepo:  docRepo,
		documents:     f.documents,
		documentTypes: &fakeDocumentTypes{types: map[pulid.ID]*documenttype.DocumentType{pod.ID: pod, bol.ID: bol}},
		uploads:       f.uploads,
		pdf:           f.pdf,
		replies:       f.drafter,
		events:        f.events,
		loads:         f.loads,
		exceptions:    f.exceptions,
		detention:     f.detention,
		now:           func() int64 { return 1_791_000_000 },
	}

	return f
}

func (f *fixture) storedAttachment() *inboundmessage.InboundAttachment {
	f.messages.mu.Lock()
	defer f.messages.mu.Unlock()

	return f.messages.attachments[f.attachment.ID]
}

func (f *fixture) storedMessage() *inboundmessage.InboundMessage {
	f.messages.mu.Lock()
	defer f.messages.mu.Unlock()

	return f.messages.messages[f.message.ID]
}

func errorFields(err error) []string {
	if multi, ok := errors.AsType[*errortypes.MultiError](err); ok {
		fields := make([]string, 0, len(multi.Errors))
		for _, item := range multi.Errors {
			fields = append(fields, item.Field)
		}

		return fields
	}
	if single, ok := errors.AsType[*errortypes.Error](err); ok {
		return []string{single.Field}
	}

	return nil
}
