package triageports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type MessageRow struct {
	ID                      pulid.ID
	OrganizationID          pulid.ID
	BusinessUnitID          pulid.ID
	MailboxID               pulid.ID
	FromAddress             string
	FromName                string
	Subject                 string
	SenderDomain            string
	Classification          inboundmessage.Classification
	Status                  inboundmessage.Status
	ReceivedAt              int64
	DueAt                   int64
	Parked                  inboundmessage.Parking
	OwnerID                 pulid.ID
	MatchedShipmentID       pulid.ID
	MatchedCustomerID       pulid.ID
	MatchedCarrierID        pulid.ID
	HandledByRuleID         pulid.ID
	HandledAction           inboundmessage.NovaAction
	ClassificationCorrected bool
	ReviewedBy              pulid.ID
	ReviewedAt              int64
	Reference               string
	PersonFiled             bool
}

func (r *MessageRow) Message() *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:                      r.ID,
		OrganizationID:          r.OrganizationID,
		BusinessUnitID:          r.BusinessUnitID,
		MailboxID:               r.MailboxID,
		FromAddress:             r.FromAddress,
		FromName:                r.FromName,
		Subject:                 r.Subject,
		SenderDomain:            r.SenderDomain,
		Classification:          r.Classification,
		Status:                  r.Status,
		ReceivedAt:              r.ReceivedAt,
		DueAt:                   r.DueAt,
		Parked:                  r.Parked,
		OwnerID:                 r.OwnerID,
		MatchedShipmentID:       r.MatchedShipmentID,
		MatchedCustomerID:       r.MatchedCustomerID,
		MatchedCarrierID:        r.MatchedCarrierID,
		HandledByRuleID:         r.HandledByRuleID,
		HandledAction:           r.HandledAction,
		ClassificationCorrected: r.ClassificationCorrected,
		ReviewedBy:              r.ReviewedBy,
		ReviewedAt:              r.ReviewedAt,
	}
}

func (r *MessageRow) SettledByRule() bool {
	return r.HandledByRuleID.IsNotNil() && r.ReviewedBy.IsNil() &&
		(r.Status == inboundmessage.StatusActioned || r.Status == inboundmessage.StatusIgnored)
}

func (r *MessageRow) SettledByPerson() bool {
	return r.ReviewedBy.IsNotNil() &&
		(r.Status == inboundmessage.StatusActioned || r.Status == inboundmessage.StatusIgnored)
}

func (r *MessageRow) Done() bool {
	return r.Status == inboundmessage.StatusActioned || r.Status == inboundmessage.StatusIgnored ||
		r.Parked != ""
}

func (r *MessageRow) Sample() *inboundmessageports.Sample {
	sample := &inboundmessageports.Sample{
		MessageID:   r.ID,
		FromName:    r.FromName,
		FromAddress: r.FromAddress,
		Subject:     r.Subject,
		ReceivedAt:  r.ReceivedAt,
		Reference:   r.Reference,
	}
	if r.Classification != "" {
		class := r.Classification
		sample.Classification = &class
	}

	return sample
}

type RuleCandidatesRequest struct {
	TenantInfo     pagination.TenantInfo
	Classification inboundmessage.Classification
	SenderScope    triage.SenderScope
	Sender         string
	MailboxID      pulid.ID
	Since          int64
	Limit          int
}

type RuleCandidates struct {
	Total int
	Rows  []*MessageRow
}

type ManualOutcomesRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
}

type ManualOutcomeRow struct {
	SenderDomain   string
	Classification inboundmessage.Classification
	Outcome        triage.ActivityKind
	Count          int
	Arrived        int
	Mailboxes      int
	MailboxID      pulid.ID
	AllCustomers   bool
	AllCarriers    bool
}

type MessageRowsRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type SenderDomainsRequest struct {
	TenantInfo pagination.TenantInfo
	Query      string
	Since      int64
	Limit      int
}

type BriefingWindow struct {
	TenantInfo pagination.TenantInfo
	DayStart   int64
	Now        int64
	Limit      int
}

type BriefingCounts struct {
	Waiting             int
	Overdue             int
	ClosedByPeopleToday int
	HandledByNovaToday  int
	ComingBack          int
	AnyMessage          bool
}

type HandledRow struct {
	MessageRow

	GroupCount int
}

type ArrivalRow struct {
	Hour          int
	HandledByNova int
	LeftForPerson int
}

type RiverRow struct {
	MessageRow

	Nova bool
}

type WaitingRequest struct {
	TenantInfo pagination.TenantInfo
	Limit      int
}

type CorrectionTotalsRequest struct {
	TenantInfo    pagination.TenantInfo
	LastWeekStart int64
	WeekStart     int64
}

type CorrectionTotals struct {
	CorrectionsThisWeek int
	ReadThisWeek        int
	CorrectedThisWeek   int
	ReadLastWeek        int
	CorrectedLastWeek   int
}

type SenderStatsRequest struct {
	TenantInfo   pagination.TenantInfo
	SenderDomain string
	Since        int64
	ExcludeID    pulid.ID
	RecentLimit  int
}

type SenderKindRow struct {
	Classification inboundmessage.Classification
	Count          int
}

type SenderStats struct {
	Messages           int
	UsualSettleSeconds *float64
	Kinds              []SenderKindRow
	CustomerID         pulid.ID
	CarrierID          pulid.ID
	Recent             []*MessageRow
}

type SenderPartyRequest struct {
	TenantInfo   pagination.TenantInfo
	Address      string
	SenderDomain string
	CustomerID   pulid.ID
	CarrierID    pulid.ID
}

type SenderParty struct {
	Role  inboundmessageports.SenderRole
	Since int64
}

type TriageInsightRepository interface {
	RuleCandidates(ctx context.Context, req *RuleCandidatesRequest) (*RuleCandidates, error)
	ManualOutcomes(ctx context.Context, req *ManualOutcomesRequest) ([]*ManualOutcomeRow, error)
	MessageRows(ctx context.Context, req *MessageRowsRequest) ([]*MessageRow, error)
	SenderDomains(ctx context.Context, req *SenderDomainsRequest) ([]string, error)
	BriefingCounts(ctx context.Context, req BriefingWindow) (*BriefingCounts, error)
	HandledByNova(ctx context.Context, req BriefingWindow) ([]*HandledRow, error)
	Arrivals(ctx context.Context, req BriefingWindow) ([]*ArrivalRow, error)
	River(ctx context.Context, req BriefingWindow) ([]*RiverRow, error)
	Waiting(ctx context.Context, req WaitingRequest) ([]*inboundmessage.InboundMessage, error)
	CorrectionTotals(ctx context.Context, req CorrectionTotalsRequest) (*CorrectionTotals, error)
	SenderStats(ctx context.Context, req *SenderStatsRequest) (*SenderStats, error)
	SenderParty(ctx context.Context, req *SenderPartyRequest) (*SenderParty, error)
}
