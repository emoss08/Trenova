package inboundreads

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

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

func waitingCount(row *inboundmessageports.InboundMessageCount) int {
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
		counts[idx].Waiting += waitingCount(&row)
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
		counts[idx].Waiting += waitingCount(&row)
	}

	slices.SortFunc(counts, func(a, b MailboxCount) int {
		return strings.Compare(a.MailboxID.String(), b.MailboxID.String())
	})

	return counts
}
