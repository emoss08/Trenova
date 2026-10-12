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
	ID                      pulid.ID                      `bun:"id"`
	OrganizationID          pulid.ID                      `bun:"organization_id"`
	BusinessUnitID          pulid.ID                      `bun:"business_unit_id"`
	MailboxID               pulid.ID                      `bun:"mailbox_id"`
	FromAddress             string                        `bun:"from_address"`
	FromName                string                        `bun:"from_name"`
	Subject                 string                        `bun:"subject"`
	SenderDomain            string                        `bun:"sender_domain"`
	Classification          inboundmessage.Classification `bun:"classification"`
	Status                  inboundmessage.Status         `bun:"status"`
	ReceivedAt              int64                         `bun:"received_at"`
	DueAt                   int64                         `bun:"due_at"`
	Parked                  inboundmessage.Parking        `bun:"parked"`
	OwnerID                 pulid.ID                      `bun:"owner_id"`
	MatchedShipmentID       pulid.ID                      `bun:"matched_shipment_id"`
	MatchedCustomerID       pulid.ID                      `bun:"matched_customer_id"`
	MatchedCarrierID        pulid.ID                      `bun:"matched_carrier_id"`
	HandledByRuleID         pulid.ID                      `bun:"handled_by_rule_id"`
	HandledAction           inboundmessage.NovaAction     `bun:"handled_action"`
	ClassificationCorrected bool                          `bun:"classification_corrected"`
	ReviewedBy              pulid.ID                      `bun:"reviewed_by"`
	ReviewedAt              int64                         `bun:"reviewed_at"`
	Reference               string                        `bun:"reference"`
	PersonFiled             bool                          `bun:"person_filed"`
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
	SenderDomain   string                        `bun:"sender_domain"`
	Classification inboundmessage.Classification `bun:"classification"`
	Outcome        triage.ActivityKind           `bun:"outcome"`
	Count          int                           `bun:"count"`
	Arrived        int                           `bun:"arrived"`
	Mailboxes      int                           `bun:"mailboxes"`
	MailboxID      pulid.ID                      `bun:"mailbox_id"`
	AllCustomers   bool                          `bun:"all_customers"`
	AllCarriers    bool                          `bun:"all_carriers"`
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
	Waiting             int  `bun:"waiting"`
	Overdue             int  `bun:"overdue"`
	ClosedByPeopleToday int  `bun:"closed_by_people_today"`
	HandledByNovaToday  int  `bun:"handled_by_nova_today"`
	ComingBack          int  `bun:"coming_back"`
	AnyMessage          bool `bun:"any_message"`
}

type HandledRow struct {
	MessageRow `bun:",extend"`

	GroupCount int `bun:"group_count"`
}

type ArrivalRow struct {
	Hour          int `bun:"hour"`
	HandledByNova int `bun:"handled_by_nova"`
	LeftForPerson int `bun:"left_for_person"`
}

type RiverRow struct {
	MessageRow `bun:",extend"`

	Nova bool `bun:"nova"`
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
	CorrectionsThisWeek int `bun:"corrections_this_week"`
	ReadThisWeek        int `bun:"read_this_week"`
	CorrectedThisWeek   int `bun:"corrected_this_week"`
	ReadLastWeek        int `bun:"read_last_week"`
	CorrectedLastWeek   int `bun:"corrected_last_week"`
}

type SenderStatsRequest struct {
	TenantInfo   pagination.TenantInfo
	SenderDomain string
	Since        int64
	ExcludeID    pulid.ID
	RecentLimit  int
}

type SenderKindRow struct {
	Classification inboundmessage.Classification `bun:"classification"`
	Count          int                           `bun:"count"`
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
	SenderDomains(ctx context.Context, req SenderDomainsRequest) ([]string, error)
	BriefingCounts(ctx context.Context, req BriefingWindow) (*BriefingCounts, error)
	HandledByNova(ctx context.Context, req BriefingWindow) ([]*HandledRow, error)
	Arrivals(ctx context.Context, req BriefingWindow) ([]*ArrivalRow, error)
	River(ctx context.Context, req BriefingWindow) ([]*RiverRow, error)
	Waiting(ctx context.Context, req WaitingRequest) ([]*inboundmessage.InboundMessage, error)
	CorrectionTotals(ctx context.Context, req CorrectionTotalsRequest) (*CorrectionTotals, error)
	SenderStats(ctx context.Context, req *SenderStatsRequest) (*SenderStats, error)
	SenderParty(ctx context.Context, req *SenderPartyRequest) (*SenderParty, error)
}
