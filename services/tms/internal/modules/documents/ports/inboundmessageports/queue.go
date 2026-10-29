package inboundmessageports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxBulkSelection      = 5000
	MaxBulkResultMessages = 100
	MaxThreadMessages     = 200
	MaxActivitiesPerRead  = 500
	HandOffBatchSize      = 500
	ReadUnreadBatchSize   = 200
	ReconcileBatchSize    = 500
	DocumentResourceType  = "shipment"
)

func (l Lane) NeedsViewer() bool {
	return l == LaneAssignedToMe
}

func (s Sort) OrDefault() Sort {
	if s.IsValid() {
		return s
	}

	return SortNewest
}

type QueueState struct {
	ID                   pulid.ID
	OrganizationID       pulid.ID
	BusinessUnitID       pulid.ID
	MailboxID            pulid.ID
	Status               inboundmessage.Status
	Classification       inboundmessage.Classification
	ReceivedAt           int64
	FromAddress          string
	FromName             string
	Subject              string
	SenderDomain         string
	ThreadKey            string
	OwnerID              pulid.ID
	DueAt                int64
	Parked               inboundmessage.Parking
	SnoozedUntil         int64
	SnoozeEvent          inboundmessage.SnoozeEvent
	SnoozeWatchID        pulid.ID
	SnoozeDocumentTypeID pulid.ID
	AwaitingUntil        int64
	MatchedShipmentID    pulid.ID
	MatchedCustomerID    pulid.ID
	MatchedCarrierID     pulid.ID
	MatchReason          string
	FailureText          string
	ReviewNote           string
	NovaRead             bool
}

func (s *QueueState) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: s.OrganizationID, BuID: s.BusinessUnitID}
}

func (s *QueueState) NeedsReview() bool {
	return s.Status == inboundmessage.StatusInReview || s.Status == inboundmessage.StatusQuarantined
}

func (s *QueueState) IsWaiting() bool {
	return s.NeedsReview() && s.Parked == ""
}

func (s *QueueState) Settled() bool {
	return s.Status == inboundmessage.StatusActioned || s.Status == inboundmessage.StatusIgnored
}

func (s *QueueState) HasTimer() bool {
	switch s.Parked {
	case inboundmessage.ParkingSnoozed:
		return s.SnoozedUntil > 0
	case inboundmessage.ParkingAwaitingReply:
		return s.AwaitingUntil > 0
	default:
		return false
	}
}

func (s *QueueState) WakeAt() int64 {
	switch s.Parked {
	case inboundmessage.ParkingSnoozed:
		return s.SnoozedUntil
	case inboundmessage.ParkingAwaitingReply:
		return s.AwaitingUntil
	default:
		return 0
	}
}

type QueueStatesRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
	ForUpdate  bool
}

type SelectInboundMessageIDsRequest struct {
	Folder *ListInboundMessagesRequest
	Limit  int
}

type DuePatchMode int

const (
	DueKeep DuePatchMode = iota
	DueClear
	DueFromTargets
)

type ParkPatch struct {
	Parking        inboundmessage.Parking
	SnoozedUntil   int64
	SnoozeEvent    inboundmessage.SnoozeEvent
	WatchShipment  bool
	DocumentTypeID pulid.ID
	By             pulid.ID
	AwaitingUntil  int64
}

type ReviewPatch struct {
	ReviewerID pulid.ID
	Note       string
	Clear      bool
}

type HandOffPatch struct {
	To pulid.ID
}

type QueuePatch struct {
	Status        inboundmessage.Status
	Review        *ReviewPatch
	Owner         *pulid.ID
	Park          *ParkPatch
	Unpark        bool
	Due           DuePatchMode
	Targets       triage.Targets
	ClearHandled  bool
	HandOff       *HandOffPatch
	ClearShipment bool
	ClearCustomer bool
	ClearCarrier  bool
	ClearReason   bool
	NovaUnread    bool
}

func (p *QueuePatch) ApplyTo(state *QueueState) {
	if p.Status != "" {
		state.Status = p.Status
	}
	if p.Review != nil && !p.Review.Clear && p.Review.Note != "" {
		state.ReviewNote = p.Review.Note
	}
	if p.Owner != nil {
		state.OwnerID = *p.Owner
	}
	switch {
	case p.Park != nil:
		p.Park.applyTo(state)
	case p.Unpark:
		state.clearParking()
	}
	switch p.Due {
	case DueClear:
		state.DueAt = 0
	case DueFromTargets:
		state.DueAt = state.ReceivedAt + int64(p.Targets.For(state.Classification))*60
	case DueKeep:
	}
	if p.ClearShipment {
		state.MatchedShipmentID = pulid.Nil
	}
	if p.ClearCustomer {
		state.MatchedCustomerID = pulid.Nil
	}
	if p.ClearCarrier {
		state.MatchedCarrierID = pulid.Nil
	}
	if p.ClearReason {
		state.MatchReason = ""
	}
	if p.NovaUnread {
		state.NovaRead = false
	}
}

func (p *ParkPatch) applyTo(state *QueueState) {
	state.clearParking()
	state.Parked = p.Parking
	if p.Parking == inboundmessage.ParkingAwaitingReply {
		state.AwaitingUntil = p.AwaitingUntil

		return
	}
	state.SnoozedUntil = p.SnoozedUntil
	state.SnoozeEvent = p.SnoozeEvent
	state.SnoozeDocumentTypeID = p.DocumentTypeID
	if p.WatchShipment {
		state.SnoozeWatchID = state.MatchedShipmentID
	}
}

func (s *QueueState) clearParking() {
	s.Parked = ""
	s.SnoozedUntil = 0
	s.SnoozeEvent = ""
	s.SnoozeWatchID = pulid.Nil
	s.SnoozeDocumentTypeID = pulid.Nil
	s.AwaitingUntil = 0
}

type UpdateQueueStateRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
	Patch      QueuePatch
	Now        int64
}

type ThreadKeysRequest struct {
	TenantInfo  pagination.TenantInfo
	MessageKeys []string
}

type ThreadCountsRequest struct {
	TenantInfo pagination.TenantInfo
	ThreadKeys []string
}

type ListThreadRequest struct {
	TenantInfo pagination.TenantInfo
	ThreadKey  string
	Limit      int
}

type AwaitingOnThreadRequest struct {
	TenantInfo pagination.TenantInfo
	ThreadKey  string
	ExcludeID  pulid.ID
}

type ListInboundMessagesByIDsRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type WakeByEventRequest struct {
	TenantInfo     pagination.TenantInfo
	Event          inboundmessage.SnoozeEvent
	WatchID        pulid.ID
	DocumentTypeID pulid.ID
	Now            int64
}

type ParkedRef struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (r ParkedRef) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: r.OrganizationID, BuID: r.BusinessUnitID}
}

type ListOverdueParkedRequest struct {
	Now   int64
	Limit int
}

type RecomputeDueRequest struct {
	TenantInfo     pagination.TenantInfo
	Classification inboundmessage.Classification
	Minutes        int
}

type UnreadInReviewRequest struct {
	TenantInfo pagination.TenantInfo
	Limit      int
}

type HandOffCandidatesRequest struct {
	TenantInfo pagination.TenantInfo
	To         pulid.ID
	Limit      int
}

type WaitingByOwnersRequest struct {
	TenantInfo pagination.TenantInfo
	OwnerIDs   []pulid.ID
}

type MailboxReceivedRequest struct {
	TenantInfo pagination.TenantInfo
	MailboxID  pulid.ID
	ReceivedAt int64
}

type MailboxTestRequest struct {
	TenantInfo pagination.TenantInfo
	MailboxID  pulid.ID
	Token      string
	At         int64
}

var ErrMailboxTestTooSoon = errors.New("a test was sent to this mailbox too recently")

type ClaimMailboxTestRequest struct {
	TenantInfo     pagination.TenantInfo
	MailboxID      pulid.ID
	Token          string
	At             int64
	LastSentBefore int64
}

type TriageQueueRepository interface {
	QueueStates(ctx context.Context, req QueueStatesRequest) ([]*QueueState, error)
	SelectIDs(ctx context.Context, req SelectInboundMessageIDsRequest) ([]pulid.ID, error)
	UpdateQueueState(ctx context.Context, req UpdateQueueStateRequest) ([]*QueueState, error)
	ListByIDs(
		ctx context.Context,
		req ListInboundMessagesByIDsRequest,
	) ([]*inboundmessage.InboundMessage, error)
	ThreadKeysByMessageKeys(ctx context.Context, req ThreadKeysRequest) (map[string]string, error)
	CountByThreadKeys(ctx context.Context, req ThreadCountsRequest) (map[string]int, error)
	ListThread(
		ctx context.Context,
		req ListThreadRequest,
	) ([]*inboundmessage.InboundMessage, error)
	AwaitingOnThread(ctx context.Context, req AwaitingOnThreadRequest) ([]*QueueState, error)
	WakeByEvent(ctx context.Context, req WakeByEventRequest) ([]*QueueState, error)
	ListOverdueParked(ctx context.Context, req ListOverdueParkedRequest) ([]ParkedRef, error)
	RecomputeDue(ctx context.Context, req RecomputeDueRequest) (int64, error)
	UnreadInReview(ctx context.Context, req UnreadInReviewRequest) ([]pulid.ID, error)
	HandOffCandidates(ctx context.Context, req HandOffCandidatesRequest) ([]pulid.ID, error)
	WaitingByOwners(ctx context.Context, req WaitingByOwnersRequest) (map[pulid.ID]int, error)
}

type TriageMailboxStateRepository interface {
	MarkReceived(ctx context.Context, req MailboxReceivedRequest) error
	ClaimTestSend(ctx context.Context, req ClaimMailboxTestRequest) (*inboundmessage.Mailbox, error)
	MarkTestSent(ctx context.Context, req MailboxTestRequest) (*inboundmessage.Mailbox, error)
	MarkTestReceived(ctx context.Context, req MailboxTestRequest) (bool, error)
}

type ListActivitiesRequest struct {
	TenantInfo pagination.TenantInfo
	MessageIDs []pulid.ID
	Limit      int
}

type TriageActivityRepository interface {
	Insert(ctx context.Context, activities []*triage.Activity) error
	ListByMessageIDs(
		ctx context.Context,
		req ListActivitiesRequest,
	) (map[pulid.ID][]*triage.Activity, error)
}

type ListViewsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

type GetViewRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ID         pulid.ID
}

type ViewWaitingCountsRequest struct {
	TenantInfo pagination.TenantInfo
	ViewIDs    []pulid.ID
}

type TriageViewRepository interface {
	List(ctx context.Context, req ListViewsRequest) ([]*triage.View, error)
	Get(ctx context.Context, req GetViewRequest) (*triage.View, error)
	Create(ctx context.Context, view *triage.View, limit int) (*triage.View, error)
	Rename(ctx context.Context, view *triage.View) (*triage.View, error)
	Delete(ctx context.Context, req GetViewRequest) error
	WaitingCounts(ctx context.Context, req ViewWaitingCountsRequest) (map[pulid.ID]int, error)
}

type SaveTargetsRequest struct {
	TenantInfo pagination.TenantInfo
	Upsert     []*triage.ResponseTarget
	Reset      []inboundmessage.Classification
}

type TriageTargetRepository interface {
	List(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*triage.ResponseTarget, error)
	Save(ctx context.Context, req SaveTargetsRequest) error
}

type TriageSettingsRepository interface {
	Get(ctx context.Context, tenantInfo pagination.TenantInfo) (*triage.Settings, error)
	Save(ctx context.Context, settings *triage.Settings) (*triage.Settings, error)
}

type InboundDocumentTypeFinder interface {
	DocumentTypeNames(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) (map[pulid.ID]string, error)
}

type TriageWakeScheduler interface {
	Enabled() bool
	ScheduleWake(ctx context.Context, tenantInfo pagination.TenantInfo, messageID pulid.ID) error
	CancelWake(ctx context.Context, messageID pulid.ID) error
	RecomputeDue(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		classifications []inboundmessage.Classification,
	) error
}

type TeachRequest struct {
	TenantInfo     pagination.TenantInfo
	ActorID        pulid.ID
	MessageID      pulid.ID
	Classification inboundmessage.Classification
	SenderDomain   string
	MailboxID      pulid.ID
	Action         inboundmessage.NovaAction
}

type TriageRuleTeacher interface {
	Teach(ctx context.Context, req TeachRequest) (*triage.Rule, error)
}

func AnnounceDocumentAttached(ctx context.Context, sink TriageEventSink, doc *document.Document) {
	if sink == nil || doc == nil || doc.ResourceType != DocumentResourceType ||
		doc.DocumentTypeID == nil || doc.DocumentTypeID.IsNil() {
		return
	}
	shipmentID, err := pulid.Parse(doc.ResourceID)
	if err != nil {
		return
	}

	sink.DocumentAttached(
		ctx,
		pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
		shipmentID,
		*doc.DocumentTypeID,
	)
}

func MessageFromState(state *QueueState) *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:                   state.ID,
		OrganizationID:       state.OrganizationID,
		BusinessUnitID:       state.BusinessUnitID,
		MailboxID:            state.MailboxID,
		Status:               state.Status,
		Classification:       state.Classification,
		ReceivedAt:           state.ReceivedAt,
		FromAddress:          state.FromAddress,
		FromName:             state.FromName,
		Subject:              state.Subject,
		SenderDomain:         state.SenderDomain,
		ThreadKey:            state.ThreadKey,
		OwnerID:              state.OwnerID,
		DueAt:                state.DueAt,
		Parked:               state.Parked,
		SnoozedUntil:         state.SnoozedUntil,
		SnoozeEvent:          state.SnoozeEvent,
		SnoozeWatchID:        state.SnoozeWatchID,
		SnoozeDocumentTypeID: state.SnoozeDocumentTypeID,
		AwaitingUntil:        state.AwaitingUntil,
		MatchedShipmentID:    state.MatchedShipmentID,
		MatchedCustomerID:    state.MatchedCustomerID,
		MatchedCarrierID:     state.MatchedCarrierID,
		MatchReason:          state.MatchReason,
		FailureText:          state.FailureText,
		ReviewNote:           state.ReviewNote,
		NovaRead:             state.NovaRead,
	}
}
