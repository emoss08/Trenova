package inboundmessageservice

import (
	"context"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const testNow = int64(1_784_131_200)

type fakeDB struct {
	kernelports.DBConnection

	txs int
}

func (f *fakeDB) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	f.txs++

	return fn(ctx, bun.Tx{})
}

type fakeQueueRepo struct {
	inboundmessageports.TriageQueueRepository

	mu          sync.Mutex
	states      map[pulid.ID]*inboundmessageports.QueueState
	threadKeys  map[string]string
	selected    []pulid.ID
	selectReq   *inboundmessageports.SelectInboundMessageIDsRequest
	updates     []inboundmessageports.UpdateQueueStateRequest
	wakeEvents  []inboundmessageports.WakeByEventRequest
	recomputed  []inboundmessageports.RecomputeDueRequest
	handOffLeft []pulid.ID
	unread      []pulid.ID
	overdue     []inboundmessageports.ParkedRef
	waiting     map[pulid.ID]int
}

func newFakeQueueRepo(states ...*inboundmessageports.QueueState) *fakeQueueRepo {
	repo := &fakeQueueRepo{
		states:     make(map[pulid.ID]*inboundmessageports.QueueState, len(states)),
		threadKeys: map[string]string{},
		waiting:    map[pulid.ID]int{},
	}
	for _, state := range states {
		repo.states[state.ID] = state
	}

	return repo
}

func cloneState(state *inboundmessageports.QueueState) *inboundmessageports.QueueState {
	copied := *state

	return &copied
}

func (f *fakeQueueRepo) QueueStates(
	_ context.Context,
	req inboundmessageports.QueueStatesRequest,
) ([]*inboundmessageports.QueueState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	states := make([]*inboundmessageports.QueueState, 0, len(req.IDs))
	for _, id := range req.IDs {
		if state, ok := f.states[id]; ok {
			states = append(states, cloneState(state))
		}
	}

	return states, nil
}

func (f *fakeQueueRepo) SelectIDs(
	_ context.Context,
	req inboundmessageports.SelectInboundMessageIDsRequest,
) ([]pulid.ID, error) {
	f.selectReq = &req

	return f.selected, nil
}

func (f *fakeQueueRepo) UpdateQueueState(
	_ context.Context,
	req inboundmessageports.UpdateQueueStateRequest,
) ([]*inboundmessageports.QueueState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.updates = append(f.updates, req)
	changed := make([]*inboundmessageports.QueueState, 0, len(req.IDs))
	for _, id := range req.IDs {
		state, ok := f.states[id]
		if !ok {
			continue
		}
		applyFakePatch(state, req.Patch, req.Now)
		changed = append(changed, cloneState(state))
	}

	return changed, nil
}

func applyFakePatch(state *inboundmessageports.QueueState, patch inboundmessageports.QueuePatch, now int64) {
	if patch.Status != "" {
		state.Status = patch.Status
	}
	if patch.Review != nil && !patch.Review.Clear && patch.Review.Note != "" {
		state.ReviewNote = patch.Review.Note
	}
	if patch.Owner != nil {
		state.OwnerID = *patch.Owner
	}
	if patch.Unpark || patch.Park != nil {
		state.Parked = ""
		state.SnoozedUntil = 0
		state.SnoozeEvent = ""
		state.SnoozeWatchID = pulid.Nil
		state.SnoozeDocumentTypeID = pulid.Nil
		state.AwaitingUntil = 0
	}
	if patch.Park != nil {
		state.Parked = patch.Park.Parking
		state.SnoozedUntil = patch.Park.SnoozedUntil
		state.SnoozeEvent = patch.Park.SnoozeEvent
		state.SnoozeDocumentTypeID = patch.Park.DocumentTypeID
		state.AwaitingUntil = patch.Park.AwaitingUntil
		if patch.Park.WatchShipment {
			state.SnoozeWatchID = state.MatchedShipmentID
		}
	}
	switch patch.Due {
	case inboundmessageports.DueClear:
		state.DueAt = 0
	case inboundmessageports.DueFromTargets:
		state.DueAt = state.ReceivedAt + int64(patch.Targets.For(state.Classification))*60
	case inboundmessageports.DueKeep:
	}
	if patch.ClearShipment {
		state.MatchedShipmentID = pulid.Nil
	}
	if patch.ClearCustomer {
		state.MatchedCustomerID = pulid.Nil
	}
	if patch.ClearCarrier {
		state.MatchedCarrierID = pulid.Nil
	}
	if patch.ClearReason {
		state.MatchReason = ""
	}
	if patch.NovaUnread {
		state.NovaRead = false
	}
	_ = now
}

func (f *fakeQueueRepo) ListByIDs(
	_ context.Context,
	req inboundmessageports.ListInboundMessagesByIDsRequest,
) ([]*inboundmessage.InboundMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	messages := make([]*inboundmessage.InboundMessage, 0, len(req.IDs))
	for _, id := range req.IDs {
		if state, ok := f.states[id]; ok {
			messages = append(messages, messageFromState(state))
		}
	}

	return messages, nil
}

func (f *fakeQueueRepo) ThreadKeysByMessageKeys(
	_ context.Context,
	req inboundmessageports.ThreadKeysRequest,
) (map[string]string, error) {
	found := make(map[string]string, len(req.MessageKeys))
	for _, key := range req.MessageKeys {
		if thread, ok := f.threadKeys[key]; ok {
			found[key] = thread
		}
	}

	return found, nil
}

func (f *fakeQueueRepo) AwaitingOnThread(
	_ context.Context,
	req inboundmessageports.AwaitingOnThreadRequest,
) ([]*inboundmessageports.QueueState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	states := make([]*inboundmessageports.QueueState, 0, 1)
	for _, state := range f.states {
		if state.ThreadKey == req.ThreadKey && state.ID != req.ExcludeID &&
			state.Parked == inboundmessage.ParkingAwaitingReply {
			states = append(states, cloneState(state))
		}
	}

	return states, nil
}

func (f *fakeQueueRepo) WakeByEvent(
	_ context.Context,
	req inboundmessageports.WakeByEventRequest,
) ([]*inboundmessageports.QueueState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.wakeEvents = append(f.wakeEvents, req)
	woken := make([]*inboundmessageports.QueueState, 0, 1)
	for _, state := range f.states {
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

func (f *fakeQueueRepo) ListOverdueParked(
	_ context.Context,
	_ inboundmessageports.ListOverdueParkedRequest,
) ([]inboundmessageports.ParkedRef, error) {
	refs := f.overdue
	f.overdue = nil

	return refs, nil
}

func (f *fakeQueueRepo) RecomputeDue(
	_ context.Context,
	req inboundmessageports.RecomputeDueRequest,
) (int64, error) {
	f.recomputed = append(f.recomputed, req)

	return 1, nil
}

func (f *fakeQueueRepo) UnreadInReview(
	_ context.Context,
	req inboundmessageports.UnreadInReviewRequest,
) ([]pulid.ID, error) {
	batch := f.unread[:min(req.Limit, len(f.unread))]
	f.unread = f.unread[len(batch):]

	return batch, nil
}

func (f *fakeQueueRepo) HandOffCandidates(
	_ context.Context,
	req inboundmessageports.HandOffCandidatesRequest,
) ([]pulid.ID, error) {
	batch := f.handOffLeft[:min(req.Limit, len(f.handOffLeft))]
	f.handOffLeft = f.handOffLeft[len(batch):]

	return batch, nil
}

func (f *fakeQueueRepo) WaitingByOwners(
	_ context.Context,
	req inboundmessageports.WaitingByOwnersRequest,
) (map[pulid.ID]int, error) {
	counts := make(map[pulid.ID]int, len(req.OwnerIDs))
	for _, id := range req.OwnerIDs {
		if count, ok := f.waiting[id]; ok {
			counts[id] = count
		}
	}

	return counts, nil
}

func (f *fakeQueueRepo) state(id pulid.ID) *inboundmessageports.QueueState {
	f.mu.Lock()
	defer f.mu.Unlock()

	return cloneState(f.states[id])
}

type fakeActivityRepo struct {
	inboundmessageports.TriageActivityRepository

	mu       sync.Mutex
	recorded []*triage.Activity
}

func (f *fakeActivityRepo) Insert(_ context.Context, activities []*triage.Activity) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.recorded = append(f.recorded, activities...)

	return nil
}

func (f *fakeActivityRepo) kinds() []triage.ActivityKind {
	f.mu.Lock()
	defer f.mu.Unlock()

	kinds := make([]triage.ActivityKind, 0, len(f.recorded))
	for _, activity := range f.recorded {
		kinds = append(kinds, activity.Kind)
	}

	return kinds
}

type fakeMailboxState struct {
	inboundmessageports.TriageMailboxStateRepository

	received     []inboundmessageports.MailboxReceivedRequest
	testsSent    []inboundmessageports.MailboxTestRequest
	testReceived []inboundmessageports.MailboxTestRequest
}

func (f *fakeMailboxState) MarkReceived(
	_ context.Context,
	req inboundmessageports.MailboxReceivedRequest,
) error {
	f.received = append(f.received, req)

	return nil
}

func (f *fakeMailboxState) MarkTestSent(
	_ context.Context,
	req inboundmessageports.MailboxTestRequest,
) (*inboundmessage.Mailbox, error) {
	f.testsSent = append(f.testsSent, req)

	return &inboundmessage.Mailbox{
		ID:         req.MailboxID,
		TestToken:  req.Token,
		TestSentAt: req.At,
	}, nil
}

func (f *fakeMailboxState) MarkTestReceived(
	_ context.Context,
	req inboundmessageports.MailboxTestRequest,
) (bool, error) {
	f.testReceived = append(f.testReceived, req)

	return true, nil
}

type fakeTargetRepo struct {
	inboundmessageports.TriageTargetRepository

	rows  []*triage.ResponseTarget
	saved []inboundmessageports.SaveTargetsRequest
}

func (f *fakeTargetRepo) List(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*triage.ResponseTarget, error) {
	return f.rows, nil
}

func (f *fakeTargetRepo) Save(_ context.Context, req inboundmessageports.SaveTargetsRequest) error {
	f.saved = append(f.saved, req)

	return nil
}

type fakeWakes struct {
	scheduled []pulid.ID
	cancelled []pulid.ID
	recompute [][]inboundmessage.Classification
}

func (f *fakeWakes) Enabled() bool { return true }

func (f *fakeWakes) ScheduleWake(_ context.Context, _ pagination.TenantInfo, id pulid.ID) error {
	f.scheduled = append(f.scheduled, id)

	return nil
}

func (f *fakeWakes) CancelWake(_ context.Context, id pulid.ID) error {
	f.cancelled = append(f.cancelled, id)

	return nil
}

func (f *fakeWakes) RecomputeDue(
	_ context.Context,
	_ pagination.TenantInfo,
	classifications []inboundmessage.Classification,
) error {
	f.recompute = append(f.recompute, slices.Clone(classifications))

	return nil
}

type queueFakes struct {
	db         *fakeDB
	queue      *fakeQueueRepo
	activities *fakeActivityRepo
	mailboxes  *fakeMailboxState
	targets    *fakeTargetRepo
	wakes      *fakeWakes
}

func withQueue(svc *Service, states ...*inboundmessageports.QueueState) (*Service, *queueFakes) {
	fakes := &queueFakes{
		db:         &fakeDB{},
		queue:      newFakeQueueRepo(states...),
		activities: &fakeActivityRepo{},
		mailboxes:  &fakeMailboxState{},
		targets:    &fakeTargetRepo{},
		wakes:      &fakeWakes{},
	}
	if svc.l == nil {
		svc.l = zap.NewNop()
	}
	svc.db = fakes.db
	svc.queueRepo = fakes.queue
	svc.activityRepo = fakes.activities
	svc.mailboxState = fakes.mailboxes
	svc.targetRepo = fakes.targets
	svc.wakes = fakes.wakes
	svc.notify = newNotifier(notifierDeps{logger: svc.l})
	svc.now = func() int64 { return testNow }

	return svc, fakes
}
