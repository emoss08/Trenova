package jurisdictionruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *service) ListOverrides(
	ctx context.Context,
	req *services.ListOverridesRequest,
) (*pagination.CursorListResult[*jurisdictionrule.Override], error) {
	return s.repo.ListOverridesConnection(ctx, &repositories.ListOverridesRequest{
		Filter: req.Filter,
		Cursor: req.Cursor,
	}, req.TenantInfo)
}

func (s *service) GetOverrideByID(
	ctx context.Context,
	overrideID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*jurisdictionrule.Override, error) {
	return s.repo.GetOverrideByID(ctx, &repositories.GetOverrideByIDRequest{
		OverrideID: overrideID,
		TenantInfo: tenantInfo,
	})
}

// validateAgainstStatute rejects an override that would loosen the jurisdiction
// rule it narrows.
//
// Resolve already refuses to apply a looser value, so this is not the safety
// property — that lives in the engine and holds for rows that arrive by any
// path. What this adds is an answer: without it the save succeeds, the operator
// believes their limit is in force, and the engine quietly uses the statutory
// one instead. A field error at write time is the difference between a setting
// that does not work and a setting the system refused.
func (s *service) validateAgainstStatute(
	ctx context.Context,
	entity *jurisdictionrule.Override,
	multiErr *errortypes.MultiError,
) {
	rules, err := s.repo.GetActiveByStateIDs(ctx, &repositories.GetJurisdictionRulesRequest{
		StateIDs: []pulid.ID{entity.StateID},
	})
	if err != nil {
		return
	}

	rule := ruleInEffect(rules, timeutils.NowUnix())
	if rule == nil {
		// Nothing in force to compare against, which is the same position as no
		// rule on file. The engine still clamps, so a looser value cannot take
		// effect; failing the save here would block a carrier from recording a
		// posture for a state nobody has configured.
		return
	}

	if entity.MaxWidthFeet != nil && *entity.MaxWidthFeet > rule.MaxWidthFeet {
		multiErr.Add("maxWidthFeet", errortypes.ErrInvalid,
			"An override can only be stricter than the state limit")
	}
	if entity.MaxHeightFeet != nil && *entity.MaxHeightFeet > rule.MaxHeightFeet {
		multiErr.Add("maxHeightFeet", errortypes.ErrInvalid,
			"An override can only be stricter than the state limit")
	}
	if entity.MaxLengthFeet != nil && *entity.MaxLengthFeet > rule.MaxLengthFeet {
		multiErr.Add("maxLengthFeet", errortypes.ErrInvalid,
			"An override can only be stricter than the state limit")
	}
	if entity.MaxWeightPounds != nil && *entity.MaxWeightPounds > rule.MaxWeightPounds {
		multiErr.Add("maxWeightPounds", errortypes.ErrInvalid,
			"An override can only be stricter than the state limit")
	}

	// Lead time is the one that runs the other way: a shorter lead time is the
	// permissive direction, because it quotes a pickup sooner than the state can
	// issue the permit.
	if entity.PermitLeadTimeDays != nil && *entity.PermitLeadTimeDays < rule.PermitLeadTimeDays {
		multiErr.Add("permitLeadTimeDays", errortypes.ErrInvalid,
			"An override can only require more lead time than the state, not less")
	}

	if entity.DaylightOnly != nil && !*entity.DaylightOnly && rule.DaylightOnly {
		multiErr.Add("daylightOnly", errortypes.ErrInvalid,
			"This state restricts movement to daylight; an override cannot lift that")
	}
	if entity.HolidayRestricted != nil && !*entity.HolidayRestricted && rule.HolidayRestricted {
		multiErr.Add("holidayRestricted", errortypes.ErrInvalid,
			"This state restricts holiday movement; an override cannot lift that")
	}
}

// ruleInEffect picks the rule the engine will actually use.
//
// GetActiveByStateIDs filters on status, not on the effective window, and
// nothing orders the query — so a state mid-transition between two rules can
// return them in either order. Validation has to narrow the same way
// resolveRules does, or it measures an override against a rule that is not in
// force and rejects a perfectly good one.
func ruleInEffect(
	rules []*jurisdictionrule.JurisdictionRule,
	at int64,
) *jurisdictionrule.JurisdictionRule {
	for _, rule := range rules {
		if rule != nil && rule.IsEffectiveAt(at) {
			return rule
		}
	}

	return nil
}

func (s *service) CreateOverride(
	ctx context.Context,
	entity *jurisdictionrule.Override,
	actor *services.RequestActor,
) (*jurisdictionrule.Override, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	s.validateAgainstStatute(ctx, entity, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateOverride(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logOverrideAction(&overrideAuditParams{
		Override:  created,
		Operation: permission.OpCreate,
		Actor:     actor,
		Current:   created,
		Comment:   "Carrier override created",
	})

	return created, nil
}

func (s *service) UpdateOverride(
	ctx context.Context,
	entity *jurisdictionrule.Override,
	actor *services.RequestActor,
) (*jurisdictionrule.Override, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	s.validateAgainstStatute(ctx, entity, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	previous, err := s.repo.GetOverrideByID(ctx, &repositories.GetOverrideByIDRequest{
		OverrideID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateOverride(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logOverrideAction(&overrideAuditParams{
		Override:  updated,
		Operation: permission.OpUpdate,
		Actor:     actor,
		Previous:  previous,
		Current:   updated,
		Comment:   "Carrier override updated",
	})

	return updated, nil
}

// DeleteOverride returns the jurisdiction to its statutory limits. That is a
// loosening, so it is audited like one.
func (s *service) DeleteOverride(
	ctx context.Context,
	overrideID pulid.ID,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) error {
	previous, err := s.repo.GetOverrideByID(ctx, &repositories.GetOverrideByIDRequest{
		OverrideID: overrideID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if err = s.repo.DeleteOverride(ctx, &repositories.DeleteOverrideRequest{
		OverrideID: overrideID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return err
	}

	s.logOverrideAction(&overrideAuditParams{
		Override:  previous,
		Operation: permission.OpDelete,
		Actor:     actor,
		Previous:  previous,
		Comment:   "Carrier override removed; this jurisdiction reverts to its statutory limits",
	})

	return nil
}

// overrideAuditParams groups what an override audit entry needs. Passed as a
// struct rather than positionally: the previous and current states share a type,
// and a transposed pair produces a diff that reads backwards.
type overrideAuditParams struct {
	Override  *jurisdictionrule.Override
	Operation permission.Operation
	Actor     *services.RequestActor
	Previous  any
	Current   any
	Comment   string
}

// logOverrideAction records an override change against the tenant that owns it.
//
// Unlike a jurisdiction rule, an override is tenant data, so the entry carries
// the organization and business unit. It is Critical for the same reason the
// rule entries are: the row decides whether a load is flagged before it leaves.
func (s *service) logOverrideAction(params *overrideAuditParams) {
	if params.Override == nil {
		return
	}

	auditservice.Record(s.auditService, s.l, &auditservice.RecordParams{
		Resource:       permission.ResourceJurisdictionRuleOverride,
		ResourceID:     params.Override.ID.String(),
		Operation:      params.Operation,
		Actor:          params.Actor.AuditActorOrSystem(),
		OrganizationID: params.Override.OrganizationID,
		BusinessUnitID: params.Override.BusinessUnitID,
		Critical:       true,
		Previous:       params.Previous,
		Current:        params.Current,
		Comment:        params.Comment,
		Metadata: map[string]any{
			"stateId": params.Override.StateID.String(),
			"reason":  params.Override.Reason,
		},
	})
}
