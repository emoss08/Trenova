package inboundmessageservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// maxReviewNoteLength bounds what a person may write on a message. It is the
// column's length, checked here so a long note is a field error rather than a
// database error the reader cannot act on.
const maxReviewNoteLength = 2000

// Counts is the inbox by lane. Waiting is the only one anybody watches, but
// the others are what make it a lane rather than a list.
type Counts struct {
	Waiting          int                   `json:"waiting"`
	WaitingOnThem    int                   `json:"waitingOnThem"`
	Snoozed          int                   `json:"snoozed"`
	AssignedToMe     int                   `json:"assignedToMe"`
	Overdue          int                   `json:"overdue"`
	Processing       int                   `json:"processing"`
	Handled          int                   `json:"handled"`
	Ignored          int                   `json:"ignored"`
	Quarantined      int                   `json:"quarantined"`
	Total            int                   `json:"total"`
	ByClassification []ClassificationCount `json:"byClassification"`
	ByMailbox        []MailboxCount        `json:"byMailbox"`
}

// ClassificationCount is one kind of mail: how much of it there is, and how
// much of that is waiting on a person.
type ClassificationCount struct {
	Classification inboundmessage.Classification `json:"classification"`
	Total          int                           `json:"total"`
	Waiting        int                           `json:"waiting"`
}

// MailboxCount is one address's share of the inbox.
type MailboxCount struct {
	MailboxID pulid.ID `json:"mailboxId"`
	Total     int      `json:"total"`
	Waiting   int      `json:"waiting"`
}

// ReviewRequest is a person's decision about a message.
type ReviewRequest struct {
	MessageID  pulid.ID
	TenantInfo pagination.TenantInfo
	ReviewerID pulid.ID
	Status     inboundmessage.Status
	Note       string
}

// LinkRequest says what a message is about, by hand.
type LinkRequest struct {
	MessageID  pulid.ID
	TenantInfo pagination.TenantInfo
	ReviewerID pulid.ID
	ShipmentID pulid.ID
	CustomerID pulid.ID
	CarrierID  pulid.ID
	Reason     string
}

func (s *Service) List(
	ctx context.Context,
	req *inboundmessageports.ListInboundMessagesRequest,
) (*pagination.CursorListResult[*inboundmessage.InboundMessage], error) {
	if err := PrepareList(req); err != nil {
		return nil, err
	}

	return s.messageRepo.ListCursor(ctx, req)
}

func PrepareList(req *inboundmessageports.ListInboundMessagesRequest) error {
	if req.Lane != "" && !req.Lane.IsValid() {
		return errortypes.NewValidationError(
			"lane",
			errortypes.ErrInvalid,
			"That folder does not exist",
		)
	}
	if req.Sort != "" && !req.Sort.IsValid() {
		return errortypes.NewValidationError("sort", errortypes.ErrInvalid,
			"Sort by due or newest")
	}
	if req.Lane.NeedsViewer() && req.ViewerID.IsNil() {
		return errortypes.NewValidationError("lane", errortypes.ErrInvalid,
			"Assigned to me needs someone signed in")
	}
	if len(req.ThreadKey) > inboundmessage.MaxThreadKeyLength {
		return errortypes.NewValidationError("threadKey", errortypes.ErrInvalid,
			"That is not a thread")
	}

	req.Sort = req.Sort.OrDefault()
	if req.ThreadKey != "" {
		req.Sort = inboundmessageports.SortNewest
	}
	if req.Filter != nil {
		req.Filter.Sort = SortFields(req.Sort)
		req.Filter.Query = strings.TrimSpace(req.Filter.Query)
	}

	return nil
}

func SortFields(sort inboundmessageports.Sort) []domaintypes.SortField {
	if sort == inboundmessageports.SortDue {
		return []domaintypes.SortField{
			{Field: "dueAt", Direction: dbtype.SortDirectionAsc},
			{Field: "receivedAt", Direction: dbtype.SortDirectionDesc},
		}
	}

	return []domaintypes.SortField{{Field: "receivedAt", Direction: dbtype.SortDirectionDesc}}
}

func (s *Service) GetByID(
	ctx context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	return s.messageRepo.GetByID(ctx, req)
}

func (s *Service) ListMailboxes(
	ctx context.Context,
	req *inboundmessageports.ListMailboxesRequest,
) (*pagination.ListResult[*inboundmessage.Mailbox], error) {
	return s.mailboxRepo.List(ctx, req)
}

func (s *Service) Counts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	viewerID pulid.ID,
) (*Counts, error) {
	rows, err := s.messageRepo.CountBreakdown(ctx, inboundmessageports.CountInboundMessagesRequest{
		TenantInfo: tenantInfo,
		ViewerID:   viewerID,
		Now:        s.now(),
	})
	if err != nil {
		return nil, err
	}

	return FoldCounts(rows), nil
}

func FoldCounts(rows []inboundmessageports.InboundMessageCount) *Counts {
	counts := &Counts{}
	for i := range rows {
		row := &rows[i]
		counts.Total += row.Count
		switch row.Status {
		case inboundmessage.StatusReceived, inboundmessage.StatusProcessing,
			inboundmessage.StatusClassified:
			counts.Processing += row.Count
		case inboundmessage.StatusActioned:
			counts.Handled += row.Count
		case inboundmessage.StatusIgnored:
			counts.Ignored += row.Count
		case inboundmessage.StatusInReview, inboundmessage.StatusQuarantined:
			foldReview(counts, row)
		}
	}
	counts.ByClassification = countByClassification(rows)
	counts.ByMailbox = countByMailbox(rows)

	return counts
}

func foldReview(counts *Counts, row *inboundmessageports.InboundMessageCount) {
	if row.Status == inboundmessage.StatusQuarantined {
		counts.Quarantined += row.Count
	}

	switch row.Parked {
	case inboundmessage.ParkingSnoozed:
		counts.Snoozed += row.Count
	case inboundmessage.ParkingAwaitingReply:
		counts.WaitingOnThem += row.Count
	default:
		counts.Waiting += row.Count
		if row.Mine {
			counts.AssignedToMe += row.Count
		}
		if row.Overdue {
			counts.Overdue += row.Count
		}
	}
}

func waitingCount(row inboundmessageports.InboundMessageCount) int {
	if (row.Status == inboundmessage.StatusInReview ||
		row.Status == inboundmessage.StatusQuarantined) && row.Parked == "" {
		return row.Count
	}

	return 0
}

// countByClassification lists every kind, in the enum's order, so a kind with
// no mail reads as empty rather than missing. Mail nothing has read yet is in
// the lanes but in no kind: it has not been told apart.
func countByClassification(rows []inboundmessageports.InboundMessageCount) []ClassificationCount {
	kinds := inboundmessage.AllClassifications()
	indexByKind := make(map[inboundmessage.Classification]int, len(kinds))
	counts := make([]ClassificationCount, len(kinds))
	for idx, kind := range kinds {
		indexByKind[kind] = idx
		counts[idx].Classification = kind
	}

	for _, row := range rows {
		idx, ok := indexByKind[row.Classification]
		if !ok {
			continue
		}
		counts[idx].Total += row.Count
		counts[idx].Waiting += waitingCount(row)
	}

	return counts
}

// countByMailbox lists only the mailboxes that have mail, in a stable order.
func countByMailbox(rows []inboundmessageports.InboundMessageCount) []MailboxCount {
	indexByMailbox := make(map[pulid.ID]int, len(rows))
	counts := make([]MailboxCount, 0, len(rows))

	for _, row := range rows {
		idx, ok := indexByMailbox[row.MailboxID]
		if !ok {
			idx = len(counts)
			indexByMailbox[row.MailboxID] = idx
			counts = append(counts, MailboxCount{MailboxID: row.MailboxID})
		}
		counts[idx].Total += row.Count
		counts[idx].Waiting += waitingCount(row)
	}

	slices.SortFunc(counts, func(a, b MailboxCount) int {
		return strings.Compare(a.MailboxID.String(), b.MailboxID.String())
	})

	return counts
}

// Review records what a person decided.
//
// Only Actioned and Ignored are a person's to choose. The rest are states the
// pipeline moves through, and letting a reader set one would put a message
// back into a queue nothing is going to pick it up from.
func (s *Service) Review(
	ctx context.Context,
	req ReviewRequest,
) (*inboundmessage.InboundMessage, error) {
	if _, err := reviewNote(&req); err != nil {
		return nil, err
	}

	message, err := s.messageRepo.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	before := queueStateOf(message)
	if err = ApplyReview(message, &req, timeutils.NowUnix()); err != nil {
		return nil, err
	}
	message.DueAt = 0

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}

	s.recordActivity(ctx, byPerson(
		triage.NewActivity(updated, settleActivityKind(updated.Status)).
			About("", updated.ReviewNote),
		req.ReviewerID,
	))
	s.afterQueueChange(ctx, req.TenantInfo, &transitionResult{
		before:  map[pulid.ID]*inboundmessageports.QueueState{updated.ID: before},
		changed: []*inboundmessageports.QueueState{queueStateOf(updated)},
	})

	return updated, nil
}

// Link says what a message is about when the reading did not work it out.
//
// The reason is required rather than optional. A hand-made link with no reason
// is indistinguishable on the page from one the classifier guessed at, and the
// whole point of storing the reason is that somebody can check it.
func (s *Service) Link(
	ctx context.Context,
	req LinkRequest,
) (*inboundmessage.InboundMessage, error) {
	if err := s.CheckLink(ctx, req); err != nil {
		return nil, err
	}

	message, err := s.messageRepo.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	ApplyLink(message, &req)

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}

	linked := byPerson(
		triage.NewActivity(updated, triage.ActivityLinked).About(updated.MatchReason, ""),
		req.ReviewerID,
	)
	linked.RecordID = pulid.FirstNotNil(req.ShipmentID, req.CustomerID, req.CarrierID)
	s.recordActivity(ctx, linked)
	s.notify.MessagesChanged(ctx, updated)

	return updated, nil
}

// CheckLink says whether a link would be accepted, without making it. A desk
// asks before proposing, so a link to a record that is not there is refused
// while the call can still be fixed rather than after a person approved it.
func (s *Service) CheckLink(ctx context.Context, req LinkRequest) error {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return errortypes.NewValidationError("reason", errortypes.ErrRequired,
			"Say why this is the right record")
	}
	if utf8.RuneCountInString(reason) > inboundmessage.MaxMatchReasonLength {
		return errortypes.NewValidationError("reason", errortypes.ErrInvalid,
			fmt.Sprintf("A reason may be at most %d characters",
				inboundmessage.MaxMatchReasonLength))
	}
	if req.ShipmentID.IsNil() && req.CustomerID.IsNil() && req.CarrierID.IsNil() {
		return errortypes.NewValidationError("shipmentId", errortypes.ErrRequired,
			"Name at least one record this message is about")
	}

	return s.verifyLinkTargets(ctx, req)
}

// verifyLinkTargets confirms each record a link names exists in the tenant.
//
// The columns carry no foreign key — a message can outlive the load it was
// about — so this is the only thing standing between a link and an id from
// another tenant, or one that was never real. A link that cannot be checked is
// refused rather than stored on trust.
func (s *Service) verifyLinkTargets(ctx context.Context, req LinkRequest) error {
	checks := []struct {
		field  string
		id     pulid.ID
		label  string
		exists func(context.Context, pagination.TenantInfo, pulid.ID) (bool, error)
	}{
		{field: "shipmentId", id: req.ShipmentID, label: "shipment"},
		{field: "customerId", id: req.CustomerID, label: "customer"},
		{field: "carrierId", id: req.CarrierID, label: "carrier"},
	}
	if s.shipments != nil {
		checks[0].exists = s.shipments.ShipmentExists
	}
	if s.parties != nil {
		checks[1].exists = s.parties.CustomerExists
		checks[2].exists = s.parties.CarrierExists
	}

	multiErr := errortypes.NewMultiError()
	for _, check := range checks {
		if check.id.IsNil() {
			continue
		}
		if check.exists == nil {
			return errortypes.NewBusinessError(
				"Linked records cannot be verified on this installation",
			)
		}

		found, err := check.exists(ctx, req.TenantInfo, check.id)
		if err != nil {
			return err
		}
		if !found {
			multiErr.Add(check.field, errortypes.ErrInvalid,
				fmt.Sprintf("No %s with that id exists", check.label))
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
