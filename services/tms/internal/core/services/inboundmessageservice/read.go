package inboundmessageservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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

// List reads the inbox newest arrival first. The cursor's own default is the
// row's creation time, which is when the pipeline wrote it, so a message a
// provider redelivered late would otherwise sit above one that arrived after it.
func (s *Service) List(
	ctx context.Context,
	req *repositories.ListInboundMessagesRequest,
) (*pagination.CursorListResult[*inboundmessage.InboundMessage], error) {
	if req.Filter != nil {
		req.Filter.Sort = []domaintypes.SortField{
			{Field: "receivedAt", Direction: dbtype.SortDirectionDesc},
		}
		req.Filter.Query = strings.TrimSpace(req.Filter.Query)
	}

	return s.messageRepo.ListCursor(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	return s.messageRepo.GetByID(ctx, req)
}

func (s *Service) ListMailboxes(
	ctx context.Context,
	req *repositories.ListMailboxesRequest,
) (*pagination.ListResult[*inboundmessage.Mailbox], error) {
	return s.mailboxRepo.List(ctx, req)
}

// Counts reads the lanes in one pass. It counts every status rather than only
// the waiting one, because a lane that is empty has to be shown as empty — a
// missing count reads as a missing lane.
func (s *Service) Counts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*Counts, error) {
	rows, err := s.messageRepo.CountBreakdown(ctx, repositories.CountInboundMessagesRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	byStatus := make(map[inboundmessage.Status]int, len(rows))
	for _, row := range rows {
		byStatus[row.Status] += row.Count
	}

	counts := &Counts{
		Waiting: byStatus[inboundmessage.StatusInReview] +
			byStatus[inboundmessage.StatusQuarantined],
		Handled:     byStatus[inboundmessage.StatusActioned],
		Ignored:     byStatus[inboundmessage.StatusIgnored],
		Quarantined: byStatus[inboundmessage.StatusQuarantined],
	}
	for _, count := range byStatus {
		counts.Total += count
	}
	counts.ByClassification = countByClassification(rows)
	counts.ByMailbox = countByMailbox(rows)

	return counts, nil
}

func waitingCount(row repositories.InboundMessageCount) int {
	if row.Status == inboundmessage.StatusInReview || row.Status == inboundmessage.StatusQuarantined {
		return row.Count
	}

	return 0
}

// countByClassification lists every kind, in the enum's order, so a kind with
// no mail reads as empty rather than missing. Mail nothing has read yet is in
// the lanes but in no kind: it has not been told apart.
func countByClassification(rows []repositories.InboundMessageCount) []ClassificationCount {
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
func countByMailbox(rows []repositories.InboundMessageCount) []MailboxCount {
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
	if req.Status != inboundmessage.StatusActioned &&
		req.Status != inboundmessage.StatusIgnored {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalid,
			"A message can be marked handled or ignored, nothing else")
	}

	note := strings.TrimSpace(req.Note)
	if len(note) > maxReviewNoteLength {
		return nil, errortypes.NewValidationError("note", errortypes.ErrInvalid,
			fmt.Sprintf("A note may be at most %d characters", maxReviewNoteLength))
	}

	message, err := s.messageRepo.GetByID(ctx, repositories.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	message.Status = req.Status
	message.ReviewedBy = req.ReviewerID
	message.ReviewedAt = timeutils.NowUnix()
	if note != "" {
		message.ReviewNote = note
	}

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}

	// Off the tower: the item stood for work owed, and this is the moment it
	// stops being owed.
	s.project(ctx, updated)

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
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errortypes.NewValidationError("reason", errortypes.ErrRequired,
			"Say why this is the right record")
	}
	if req.ShipmentID.IsNil() && req.CustomerID.IsNil() && req.CarrierID.IsNil() {
		return nil, errortypes.NewValidationError("shipmentId", errortypes.ErrRequired,
			"Name at least one record this message is about")
	}

	message, err := s.messageRepo.GetByID(ctx, repositories.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	message.MatchedShipmentID = req.ShipmentID
	message.MatchedCustomerID = req.CustomerID
	message.MatchedCarrierID = req.CarrierID
	message.MatchReason = reason
	message.ReviewedBy = req.ReviewerID

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}

	// Linking does not settle the message — somebody still has to say what to
	// do about it — so the tower item is refreshed rather than resolved, and
	// now carries the reason the link was made.
	s.project(ctx, updated)

	return updated, nil
}
