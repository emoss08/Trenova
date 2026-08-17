package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// GetJurisdictionRulesRequest selects active rules for a set of jurisdictions.
//
// There is no TenantInfo and no point-in-time here, and both absences are
// deliberate. These rows are global, so there is no tenant to scope by; and the
// effective window is applied by the caller rather than the query, because the
// permit engine and override validation need to agree on which rule is in force
// and that decision belongs in one place — see ruleInEffect and resolveRules.
//
// Carrying either as an ignored field would be worse than omitting it: a caller
// reading `At` would reasonably assume the query filtered on it.
type GetJurisdictionRulesRequest struct {
	StateIDs []pulid.ID
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

type ListOverridesRequest struct {
	Filter *pagination.QueryOptions
	Cursor pagination.CursorInfo
}

type GetOverrideByIDRequest struct {
	OverrideID pulid.ID
	TenantInfo pagination.TenantInfo
}

type DeleteOverrideRequest struct {
	OverrideID pulid.ID
	TenantInfo pagination.TenantInfo
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

	// Overrides are tenant data, unlike the rules they narrow, so every method
	// below scopes by organization and business unit.
	ListOverridesConnection(
		ctx context.Context,
		req *ListOverridesRequest,
		tenantInfo pagination.TenantInfo,
	) (*pagination.CursorListResult[*jurisdictionrule.Override], error)

	GetOverrideByID(
		ctx context.Context,
		req *GetOverrideByIDRequest,
	) (*jurisdictionrule.Override, error)

	CreateOverride(
		ctx context.Context,
		entity *jurisdictionrule.Override,
	) (*jurisdictionrule.Override, error)

	UpdateOverride(
		ctx context.Context,
		entity *jurisdictionrule.Override,
	) (*jurisdictionrule.Override, error)

	DeleteOverride(ctx context.Context, req *DeleteOverrideRequest) error
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
	// ExcludeSuperseded drops rows from prior derivations. The current
	// requirement set is what the panel and the dispatch check consume;
	// superseded rows are history and belong to the audit trail.
	ExcludeSuperseded bool
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
