package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetJurisdictionRulesRequest struct {
	TenantInfo pagination.TenantInfo
	StateIDs   []pulid.ID
	At         int64
}

// ListJurisdictionRulesRequest drives the admin table.
//
// There is no TenantInfo here, and its absence is the point: jurisdiction_rules
// carries no organization or business unit column because the legal width of a
// load in Nebraska does not vary by carrier. Adding a tenant filter would imply
// an isolation this table does not have.
type ListJurisdictionRulesRequest struct {
	Filter            *pagination.QueryOptions
	VerificationState jurisdictionrule.VerificationState
	Status            jurisdictionrule.Status
}

type ListJurisdictionRuleConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"-"`
}

type GetJurisdictionRuleByIDRequest struct {
	RuleID           pulid.ID
	ExpandThresholds bool
}

type VerifyJurisdictionRuleRequest struct {
	RuleID     pulid.ID
	VerifiedAt int64
	SourceNote string
	SourceURL  string
	State      jurisdictionrule.VerificationState
}

type JurisdictionRuleRepository interface {
	GetActiveByStateIDs(
		ctx context.Context,
		req *GetJurisdictionRulesRequest,
	) ([]*jurisdictionrule.JurisdictionRule, error)

	GetOverrides(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*jurisdictionrule.Override, error)

	List(
		ctx context.Context,
		req *ListJurisdictionRulesRequest,
	) (*pagination.ListResult[*jurisdictionrule.JurisdictionRule], error)

	ListConnection(
		ctx context.Context,
		req *ListJurisdictionRuleConnectionRequest,
	) (*pagination.CursorListResult[*jurisdictionrule.JurisdictionRule], error)

	GetByID(
		ctx context.Context,
		req *GetJurisdictionRuleByIDRequest,
	) (*jurisdictionrule.JurisdictionRule, error)

	// Create and Update write globally visible reference data. Every tenant on
	// the platform reads the result, so authorization for these is not an
	// ordinary tenant-admin concern.
	Create(
		ctx context.Context,
		entity *jurisdictionrule.JurisdictionRule,
	) (*jurisdictionrule.JurisdictionRule, error)

	Update(
		ctx context.Context,
		entity *jurisdictionrule.JurisdictionRule,
	) (*jurisdictionrule.JurisdictionRule, error)

	// Verify moves a row's verification state without touching its limits, so
	// confirming a row against the statute cannot silently alter it.
	Verify(
		ctx context.Context,
		req *VerifyJurisdictionRuleRequest,
	) (*jurisdictionrule.JurisdictionRule, error)
}

type JurisdictionRuleCacheRepository interface {
	GetResolved(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (map[pulid.ID]*jurisdictionrule.Resolved, error)

	SetResolved(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		resolved map[pulid.ID]*jurisdictionrule.Resolved,
	) error

	Invalidate(ctx context.Context, tenantInfo pagination.TenantInfo) error
}

type ListPermitsRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	Status     string
}

type GetPermitByIDRequest struct {
	PermitID   pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListRequirementsRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	Status     string
}

type ReplaceRequirementsRequest struct {
	TenantInfo   pagination.TenantInfo
	ShipmentID   pulid.ID
	Requirements []*permit.Requirement
}

type ResolveRouteStatesRequest struct {
	TenantInfo  pagination.TenantInfo
	LocationIDs []pulid.ID
}

type WaiveRequirementRepoRequest struct {
	TenantInfo    pagination.TenantInfo
	RequirementID pulid.ID
	WaivedByID    pulid.ID
	Reason        string
}

type PermitRepository interface {
	ListByShipment(
		ctx context.Context,
		req *ListPermitsRequest,
	) ([]*permit.Permit, error)

	GetByID(ctx context.Context, req *GetPermitByIDRequest) (*permit.Permit, error)

	Create(ctx context.Context, entity *permit.Permit) (*permit.Permit, error)

	Update(ctx context.Context, entity *permit.Permit) (*permit.Permit, error)

	ListRequirements(
		ctx context.Context,
		req *ListRequirementsRequest,
	) ([]*permit.Requirement, error)

	// ReplaceForShipment supersedes the prior derivation and inserts the new one
	// in a single transaction, so a shipment is never observed with a mix of two
	// derivations.
	ReplaceForShipment(ctx context.Context, req *ReplaceRequirementsRequest) error

	// ResolveRouteStates maps stop locations to jurisdictions without depending
	// on the Location relation being hydrated on the mutation path.
	ResolveRouteStates(
		ctx context.Context,
		req *ResolveRouteStatesRequest,
	) (map[pulid.ID]pulid.ID, error)

	WaiveRequirement(
		ctx context.Context,
		req *WaiveRequirementRepoRequest,
	) (*permit.Requirement, error)

	FindHoldReasonByCode(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		code string,
	) (*holdreason.HoldReason, error)
}
