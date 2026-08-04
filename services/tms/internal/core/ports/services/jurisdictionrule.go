package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListJurisdictionRulesRequest struct {
	Filter            *pagination.QueryOptions
	VerificationState jurisdictionrule.VerificationState
	Status            jurisdictionrule.Status
}

type VerifyJurisdictionRuleRequest struct {
	RuleID     pulid.ID
	State      jurisdictionrule.VerificationState
	SourceNote string
	SourceURL  string
	Actor      *RequestActor
}

// JurisdictionRuleService maintains the shared oversize reference data the
// permit engine derives from.
//
// Every method here writes or reads data that is global to the platform:
// jurisdiction_rules carries no organization column, because the legal width of
// a load in a given state does not vary by carrier. A carrier that needs a
// stricter posture records a jurisdiction_rule_override instead, which is
// tenant-scoped and requires a reason.
//
// That makes writes here materially different from ordinary tenant
// administration — a bad edit changes what every organization on the platform
// is told is legal.
type JurisdictionRuleService interface {
	List(
		ctx context.Context,
		req *ListJurisdictionRulesRequest,
	) (*pagination.ListResult[*jurisdictionrule.JurisdictionRule], error)

	GetByID(
		ctx context.Context,
		ruleID pulid.ID,
	) (*jurisdictionrule.JurisdictionRule, error)

	Create(
		ctx context.Context,
		entity *jurisdictionrule.JurisdictionRule,
		actor *RequestActor,
	) (*jurisdictionrule.JurisdictionRule, error)

	Update(
		ctx context.Context,
		entity *jurisdictionrule.JurisdictionRule,
		actor *RequestActor,
	) (*jurisdictionrule.JurisdictionRule, error)

	Verify(
		ctx context.Context,
		req *VerifyJurisdictionRuleRequest,
	) (*jurisdictionrule.JurisdictionRule, error)
}
