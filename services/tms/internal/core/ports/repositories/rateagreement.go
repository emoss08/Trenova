package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListRateAgreementRequest struct {
	Filter    *pagination.QueryOptions `json:"filter"`
	PartyType rateagreement.PartyType  `json:"partyType"`
	Status    rateagreement.Status     `json:"status"`
}

type ListRateAgreementConnectionRequest struct {
	Filter               *pagination.QueryOptions `json:"filter"`
	Cursor               pagination.CursorInfo    `json:"-"`
	RateAgreementColumns []string                 `json:"-"`
}

type GetRateAgreementByIDRequest struct {
	RateAgreementID pulid.ID              `json:"rateAgreementId"`
	TenantInfo      pagination.TenantInfo `json:"-"`
	IncludeRules    bool                  `json:"includeRules"`
	IncludeChildren bool                  `json:"includeChildren"`
	// AsOf narrows the loaded rules to those effective at a moment. Zero loads
	// every rule, which is what an editor needs; a rating needs only the live
	// ones.
	AsOf int64 `json:"asOf"`
}

// ResolveRateRulesRequest fetches the rules that could price one lane.
//
// PartyIDs is a list rather than a single value so rate shopping can ask about
// every candidate carrier in one round trip instead of one query per carrier.
//
// LaneKeys are the keys the shipment could match, produced by rategeo. The
// database probes its index once per key, so the cost tracks the number of
// matching rules rather than the size of the table.
type ResolveRateRulesRequest struct {
	TenantInfo pagination.TenantInfo   `json:"-"`
	PartyType  rateagreement.PartyType `json:"partyType"`
	PartyIDs   []pulid.ID              `json:"partyIds"`
	LaneKeys   []string                `json:"laneKeys"`
	AsOf       int64                   `json:"asOf"`

	// The coordinates let radius lanes be picked up by a second, much smaller
	// geospatial query and unioned with the keyed results.
	OriginLatitude       *float64 `json:"originLatitude"`
	OriginLongitude      *float64 `json:"originLongitude"`
	DestinationLatitude  *float64 `json:"destinationLatitude"`
	DestinationLongitude *float64 `json:"destinationLongitude"`

	// Limit bounds the candidate set. Exceeding it is a data quality problem
	// the quote should surface rather than something to scan through.
	Limit int `json:"limit"`
}

// ResolveRateRulesResult carries the candidates and whether the fetch was
// capped, so a trace can say "we looked at two hundred of four hundred" rather
// than silently implying it saw everything.
type ResolveRateRulesResult struct {
	Rules  []*rateagreement.RateAgreementRule
	Total  int
	Capped bool
}

type ListRateAgreementRulesRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	RateAgreementID pulid.ID              `json:"rateAgreementId"`
	AsOf            int64                 `json:"asOf"`
	IncludeInactive bool                  `json:"includeInactive"`
}

type GetRateAgreementRuleByIDRequest struct {
	RuleID     pulid.ID              `json:"ruleId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

// AmendRateAgreementRulesRequest closes out the rules a change replaces and
// inserts their successors in one transaction.
//
// This is the single primitive behind a general rate increase, a rate sheet
// import and a simulated change, which is why rules carry their own effective
// windows rather than being copied into an agreement version.
type AmendRateAgreementRulesRequest struct {
	TenantInfo      pagination.TenantInfo              `json:"-"`
	RateAgreementID pulid.ID                           `json:"rateAgreementId"`
	EffectiveFrom   int64                              `json:"effectiveFrom"`
	SupersededIDs   []pulid.ID                         `json:"supersededIds"`
	Rules           []*rateagreement.RateAgreementRule `json:"rules"`
}

type ListRateAgreementVersionsRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	RateAgreementID pulid.ID              `json:"rateAgreementId"`
	Limit           int                   `json:"limit"`
	Offset          int                   `json:"offset"`
}

type GetEffectiveAgreementVersionRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	RateAgreementID pulid.ID              `json:"rateAgreementId"`
	AsOf            int64                 `json:"asOf"`
}

type RateAgreementRepository interface {
	List(
		ctx context.Context,
		req *ListRateAgreementRequest,
	) (*pagination.ListResult[*rateagreement.RateAgreement], error)
	ListConnection(
		ctx context.Context,
		req *ListRateAgreementConnectionRequest,
	) (*pagination.CursorListResult[*rateagreement.RateAgreement], error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*rateagreement.RateAgreement], error)
	GetByID(
		ctx context.Context,
		req *GetRateAgreementByIDRequest,
	) (*rateagreement.RateAgreement, error)
	Create(
		ctx context.Context,
		entity *rateagreement.RateAgreement,
	) (*rateagreement.RateAgreement, error)
	Update(
		ctx context.Context,
		entity *rateagreement.RateAgreement,
	) (*rateagreement.RateAgreement, error)

	ResolveRules(
		ctx context.Context,
		req *ResolveRateRulesRequest,
	) (*ResolveRateRulesResult, error)
	ListRules(
		ctx context.Context,
		req *ListRateAgreementRulesRequest,
	) ([]*rateagreement.RateAgreementRule, error)
	GetRuleByID(
		ctx context.Context,
		req *GetRateAgreementRuleByIDRequest,
	) (*rateagreement.RateAgreementRule, error)
	AmendRules(ctx context.Context, req *AmendRateAgreementRulesRequest) error

	ListVersions(
		ctx context.Context,
		req *ListRateAgreementVersionsRequest,
	) (*pagination.ListResult[*rateagreement.RateAgreementVersion], error)
	GetEffectiveVersion(
		ctx context.Context,
		req *GetEffectiveAgreementVersionRequest,
	) (*rateagreement.RateAgreementVersion, error)
	CreateVersion(
		ctx context.Context,
		version *rateagreement.RateAgreementVersion,
	) (*rateagreement.RateAgreementVersion, error)
}
