package inboundqueuetest

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const TestNow = int64(1_784_131_200)

type FakeDB struct {
	Txs int
}

func (f *FakeDB) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	f.Txs++

	return fn(ctx)
}

type FakeQueueRepo struct {
	inboundmessageports.TriageQueueRepository

	mu          sync.Mutex
	States      map[pulid.ID]*inboundmessageports.QueueState
	ThreadKeys  map[string]string
	Selected    []pulid.ID
	SelectReq   *inboundmessageports.SelectInboundMessageIDsRequest
	Updates     []inboundmessageports.UpdateQueueStateRequest
	WakeEvents  []inboundmessageports.WakeByEventRequest
	Recomputed  []inboundmessageports.RecomputeDueRequest
	HandOffLeft []pulid.ID
	Unread      []pulid.ID
	Overdue     []inboundmessageports.ParkedRef
	Waiting     map[pulid.ID]int
}

func NewFakeQueueRepo(states ...*inboundmessageports.QueueState) *FakeQueueRepo {
	repo := &FakeQueueRepo{
		States:     make(map[pulid.ID]*inboundmessageports.QueueState, len(states)),
		ThreadKeys: map[string]string{},
		Waiting:    map[pulid.ID]int{},
	}
	for _, state := range states {
		repo.States[state.ID] = state
	}

	return repo
}

func cloneState(state *inboundmessageports.QueueState) *inboundmessageports.QueueState {
	copied := *state

	return &copied
}

func (f *FakeQueueRepo) QueueStates(
	_ context.Context,
	req inboundmessageports.QueueStatesRequest,
) ([]*inboundmessageports.QueueState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	states := make([]*inboundmessageports.QueueState, 0, len(req.IDs))
	for _, id := range req.IDs {
		if state, ok := f.States[id]; ok {
			states = append(states, cloneState(state))
		}
	}

	return states, nil
}

func (f *FakeQueueRepo) SelectIDs(
	_ context.Context,
	req inboundmessageports.SelectInboundMessageIDsRequest,
) ([]pulid.ID, error) {
	f.SelectReq = &req

	return f.Selected, nil
}

func (f *FakeQueueRepo) UpdateQueueState(
	_ context.Context,
	req inboundmessageports.UpdateQueueStateRequest,
) ([]*inboundmessageports.QueueState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Updates = append(f.Updates, req)
	changed := make([]*inboundmessageports.QueueState, 0, len(req.IDs))
	for _, id := range req.IDs {
		state, ok := f.States[id]
		if !ok {
			continue
		}
		applyFakePatch(state, req.Patch, req.Now)
		changed = append(changed, cloneState(state))
	}

	return changed, nil
}

func applyFakePatch(
	state *inboundmessageports.QueueState,
	patch inboundmessageports.QueuePatch,
	_ int64,
) {
	patch.ApplyTo(state)
}

func (f *FakeQueueRepo) ListByIDs(
	_ context.Context,
	req inboundmessageports.ListInboundMessagesByIDsRequest,
) ([]*inboundmessage.InboundMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	messages := make([]*inboundmessage.InboundMessage, 0, len(req.IDs))
	for _, id := range req.IDs {
		if state, ok := f.States[id]; ok {
			messages = append(messages, inboundmessageports.MessageFromState(state))
		}
	}

	return messages, nil
}

func (f *FakeQueueRepo) ThreadKeysByMessageKeys(
	_ context.Context,
	req inboundmessageports.ThreadKeysRequest,
) (map[string]string, error) {
	found := make(map[string]string, len(req.MessageKeys))
	for _, key := range req.MessageKeys {
		if thread, ok := f.ThreadKeys[key]; ok {
			found[key] = thread
		}
	}

	return found, nil
}

func (f *FakeQueueRepo) AwaitingOnThread(
	_ context.Context,
	req inboundmessageports.AwaitingOnThreadRequest,
) ([]*inboundmessageports.QueueState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	states := make([]*inboundmessageports.QueueState, 0, 1)
	for _, state := range f.States {
		if state.ThreadKey == req.ThreadKey && state.ID != req.ExcludeID &&
			state.Parked == inboundmessage.ParkingAwaitingReply {
			states = append(states, cloneState(state))
		}
	}

	return states, nil
}

func (f *FakeQueueRepo) WakeByEvent(
	_ context.Context,
	req inboundmessageports.WakeByEventRequest,
) ([]*inboundmessageports.QueueState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.WakeEvents = append(f.WakeEvents, req)
	woken := make([]*inboundmessageports.QueueState, 0, 1)
	for _, state := range f.States {
		if state.Parked != inboundmessage.ParkingSnoozed || state.SnoozeEvent != req.Event ||
			state.SnoozeWatchID != req.WatchID {
			continue
		}
		if req.DocumentTypeID.IsNotNil() && state.SnoozeDocumentTypeID != req.DocumentTypeID {
			continue
		}
		applyFakePatch(state, inboundmessageports.QueuePatch{Unpark: true}, req.Now)
		woken = append(woken, cloneState(state))
	}

	return woken, nil
}

func (f *FakeQueueRepo) ListOverdueParked(
	_ context.Context,
	_ inboundmessageports.ListOverdueParkedRequest,
) ([]inboundmessageports.ParkedRef, error) {
	refs := f.Overdue
	f.Overdue = nil

	return refs, nil
}

func (f *FakeQueueRepo) RecomputeDue(
	_ context.Context,
	req inboundmessageports.RecomputeDueRequest,
) (int64, error) {
	f.Recomputed = append(f.Recomputed, req)

	return 1, nil
}

func (f *FakeQueueRepo) UnreadInReview(
	_ context.Context,
	req inboundmessageports.UnreadInReviewRequest,
) ([]pulid.ID, error) {
	batch := f.Unread[:min(req.Limit, len(f.Unread))]
	f.Unread = f.Unread[len(batch):]

	return batch, nil
}

func (f *FakeQueueRepo) HandOffCandidates(
	_ context.Context,
	req inboundmessageports.HandOffCandidatesRequest,
) ([]pulid.ID, error) {
	batch := f.HandOffLeft[:min(req.Limit, len(f.HandOffLeft))]
	f.HandOffLeft = f.HandOffLeft[len(batch):]

	return batch, nil
}

func (f *FakeQueueRepo) WaitingByOwners(
	_ context.Context,
	req inboundmessageports.WaitingByOwnersRequest,
) (map[pulid.ID]int, error) {
	counts := make(map[pulid.ID]int, len(req.OwnerIDs))
	for _, id := range req.OwnerIDs {
		if count, ok := f.Waiting[id]; ok {
			counts[id] = count
		}
	}

	return counts, nil
}

func (f *FakeQueueRepo) State(id pulid.ID) *inboundmessageports.QueueState {
	f.mu.Lock()
	defer f.mu.Unlock()

	return cloneState(f.States[id])
}

type FakeActivityRepo struct {
	inboundmessageports.TriageActivityRepository

	mu       sync.Mutex
	Recorded []*triage.Activity
}

func (f *FakeActivityRepo) Insert(_ context.Context, activities []*triage.Activity) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Recorded = append(f.Recorded, activities...)

	return nil
}

func (f *FakeActivityRepo) Kinds() []triage.ActivityKind {
	f.mu.Lock()
	defer f.mu.Unlock()

	kinds := make([]triage.ActivityKind, 0, len(f.Recorded))
	for _, activity := range f.Recorded {
		kinds = append(kinds, activity.Kind)
	}

	return kinds
}

type FakeMailboxState struct {
	inboundmessageports.TriageMailboxStateRepository

	Received     []inboundmessageports.MailboxReceivedRequest
	Claims       []inboundmessageports.ClaimMailboxTestRequest
	TestsSent    []inboundmessageports.MailboxTestRequest
	TestReceived []inboundmessageports.MailboxTestRequest
	LastTestAt   int64
}

func (f *FakeMailboxState) MarkReceived(
	_ context.Context,
	req inboundmessageports.MailboxReceivedRequest,
) error {
	f.Received = append(f.Received, req)

	return nil
}

func (f *FakeMailboxState) ClaimTestSend(
	_ context.Context,
	req inboundmessageports.ClaimMailboxTestRequest,
) (*inboundmessage.Mailbox, error) {
	if f.LastTestAt != 0 && f.LastTestAt > req.LastSentBefore {
		return nil, inboundmessageports.ErrMailboxTestTooSoon
	}
	f.Claims = append(f.Claims, req)
	f.LastTestAt = req.At

	return &inboundmessage.Mailbox{
		ID:         req.MailboxID,
		TestToken:  req.Token,
		TestSentAt: req.At,
	}, nil
}

func (f *FakeMailboxState) MarkTestSent(
	_ context.Context,
	req inboundmessageports.MailboxTestRequest,
) (*inboundmessage.Mailbox, error) {
	f.TestsSent = append(f.TestsSent, req)
	f.LastTestAt = req.At

	return &inboundmessage.Mailbox{
		ID:         req.MailboxID,
		TestToken:  req.Token,
		TestSentAt: req.At,
	}, nil
}

func (f *FakeMailboxState) MarkTestReceived(
	_ context.Context,
	req inboundmessageports.MailboxTestRequest,
) (bool, error) {
	f.TestReceived = append(f.TestReceived, req)

	return true, nil
}

type FakeTargetRepo struct {
	inboundmessageports.TriageTargetRepository

	Rows  []*triage.ResponseTarget
	Saved []inboundmessageports.SaveTargetsRequest
}

func (f *FakeTargetRepo) List(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*triage.ResponseTarget, error) {
	return f.Rows, nil
}

func (f *FakeTargetRepo) Save(_ context.Context, req inboundmessageports.SaveTargetsRequest) error {
	f.Saved = append(f.Saved, req)

	return nil
}

type FakeWakes struct {
	Scheduled []pulid.ID
	Cancelled []pulid.ID
	Recompute [][]inboundmessage.Classification
}

func (f *FakeWakes) Enabled() bool { return true }

func (f *FakeWakes) ScheduleWake(_ context.Context, _ pagination.TenantInfo, id pulid.ID) error {
	f.Scheduled = append(f.Scheduled, id)

	return nil
}

func (f *FakeWakes) CancelWake(_ context.Context, id pulid.ID) error {
	f.Cancelled = append(f.Cancelled, id)

	return nil
}

func (f *FakeWakes) RecomputeDue(
	_ context.Context,
	_ pagination.TenantInfo,
	classifications []inboundmessage.Classification,
) error {
	f.Recompute = append(f.Recompute, slices.Clone(classifications))

	return nil
}

type QueueFakes struct {
	DB         *FakeDB
	Queue      *FakeQueueRepo
	Activities *FakeActivityRepo
	Mailboxes  *FakeMailboxState
	Targets    *FakeTargetRepo
	Wakes      *FakeWakes
}

// plainSecrets stands in for the key manager. The secret this decrypts is
// already the base64 signing key, so the signature maths is exercised for real
// while the envelope encryption is somebody else's tested concern.
type PlainSecrets struct{}

func (PlainSecrets) EncryptStringWithAAD(value string, _ encryptionservice.AAD) (string, error) {
	return SealedPrefix + value, nil
}

func (PlainSecrets) DecryptStringWithLegacy(
	value string,
	_ encryptionservice.AAD,
) (string, bool, error) {
	return strings.TrimPrefix(value, SealedPrefix), false, nil
}

type StubShipments struct {
	ByReference map[string]pulid.ID
	Existing    map[pulid.ID]bool
	Err         error
	Asked       []string
}

func (s *StubShipments) ShipmentExists(
	_ context.Context, _ pagination.TenantInfo, id pulid.ID,
) (bool, error) {
	return s.Existing[id], s.Err
}

func (s *StubShipments) FindByReference(
	_ context.Context, _ pagination.TenantInfo, reference string,
) (pulid.ID, bool, error) {
	s.Asked = append(s.Asked, reference)
	if s.Err != nil {
		return pulid.Nil, false, s.Err
	}
	id, ok := s.ByReference[reference]

	return id, ok, nil
}

type StubParties struct {
	Customers map[string]pulid.ID
	Carriers  map[string]pulid.ID
	Existing  map[pulid.ID]bool
	Err       error
}

func (s *StubParties) CustomerExists(
	_ context.Context, _ pagination.TenantInfo, id pulid.ID,
) (bool, error) {
	return s.Existing[id], s.Err
}

func (s *StubParties) CarrierExists(
	_ context.Context, _ pagination.TenantInfo, id pulid.ID,
) (bool, error) {
	return s.Existing[id], s.Err
}

func (s *StubParties) FindCustomerByEmail(
	_ context.Context, _ pagination.TenantInfo, address string,
) (pulid.ID, bool, error) {
	if s.Err != nil {
		return pulid.Nil, false, s.Err
	}
	id, ok := s.Customers[address]

	return id, ok, nil
}

func (s *StubParties) FindCarrierByEmail(
	_ context.Context, _ pagination.TenantInfo, address string,
) (pulid.ID, bool, error) {
	if s.Err != nil {
		return pulid.Nil, false, s.Err
	}
	id, ok := s.Carriers[address]

	return id, ok, nil
}

type RecordingRealtime struct {
	realtimeports.RealtimeService

	Published []*realtimeports.PublishResourceInvalidationRequest
}

func (r *RecordingRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *realtimeports.PublishResourceInvalidationRequest,
) error {
	r.Published = append(r.Published, req)
	return nil
}

type SettleRepo struct {
	inboundmessageports.InboundMessageRepository

	Message *inboundmessage.InboundMessage
	GetErr  error
	Saved   *inboundmessage.InboundMessage
	Writes  int
}

func (s *SettleRepo) GetByID(
	_ context.Context, _ inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}

	return s.Message, nil
}

func (s *SettleRepo) Update(
	_ context.Context, entity *inboundmessage.InboundMessage,
) (*inboundmessage.InboundMessage, error) {
	s.Writes++
	s.Saved = entity

	return entity, nil
}

func Staged(
	status inboundmessage.Status,
	policy inboundmessage.ReviewPolicy,
) *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:          pulid.MustNew("imsg_"),
		Status:      status,
		FromAddress: "dispatch@bigshipper.com",
		Subject:     "Load 88213 tender",
		TextBody:    "Please confirm pickup.",
		Mailbox: &inboundmessage.Mailbox{
			ID:           pulid.MustNew("imbx_"),
			ReviewPolicy: policy,
			Status:       inboundmessage.MailboxActive,
		},
	}
}

func NewQueueFakes(states ...*inboundmessageports.QueueState) *QueueFakes {
	return &QueueFakes{
		DB:         &FakeDB{},
		Queue:      NewFakeQueueRepo(states...),
		Activities: &FakeActivityRepo{},
		Mailboxes:  &FakeMailboxState{},
		Targets:    &FakeTargetRepo{},
		Wakes:      &FakeWakes{},
	}
}

func TestMailbox() *inboundmessage.Mailbox {
	return &inboundmessage.Mailbox{
		ID:             pulid.MustNew("imbx_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Address:        "tenders@acme-logistics.com",
		Provider:       inboundmessage.ProviderResend,
		TokenHash:      hashutils.SHA256Hex(TestToken),
		SigningSecret:  TestSecret,
		ReviewPolicy:   inboundmessage.ReviewAlways,
		Status:         inboundmessage.MailboxActive,
	}
}

// sealedPrefix marks a value the stub has "sealed", so a test can tell a
// secret that went through the keeper from one stored in the clear.
const SealedPrefix = "sealed:"

type ContentRepo struct {
	inboundmessageports.InboundMessageRepository

	mu sync.Mutex

	Message            *inboundmessage.InboundMessage
	Saved              *inboundmessage.InboundMessage
	Writes             int
	AskedForAttachment bool
}

func (r *ContentRepo) GetByID(
	_ context.Context, req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	r.AskedForAttachment = r.AskedForAttachment || req.IncludeAttachments

	return r.Message, nil
}

func (r *ContentRepo) Update(
	ctx context.Context, entity *inboundmessage.InboundMessage,
) (*inboundmessage.InboundMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Writes++
	copied := *entity
	copied.Attachments = nil
	copied.Mailbox = nil
	r.Saved = &copied

	return &copied, nil
}

func (r *ContentRepo) Snapshot() (int, *inboundmessage.InboundMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Writes, r.Saved
}

func (r *ContentRepo) ListAttachments(
	ctx context.Context, _ pulid.ID, _ pagination.TenantInfo,
) ([]*inboundmessage.InboundAttachment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.Message.Attachments, nil
}

func (r *ContentRepo) UpdateAttachment(
	_ context.Context, entity *inboundmessage.InboundAttachment,
) (*inboundmessage.InboundAttachment, error) {
	return entity, nil
}

var QueueTenant = pagination.TenantInfo{
	OrgID: "org_01J0000000000000000000000A",
	BuID:  "bu_01J00000000000000000000000B",
}

const (
	TestToken  = "imbx_live_2f8a91c0"
	TestSecret = "c2lnbmluZy1rZXktZm9yLXRoZS10ZXN0cw=="
)
