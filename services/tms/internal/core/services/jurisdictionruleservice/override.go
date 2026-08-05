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
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
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
	if err != nil || len(rules) == 0 || rules[0] == nil {
		// No statute on file to compare against. The engine still clamps, so a
		// looser value cannot take effect; failing the save here would block a
		// carrier from recording a posture for a state nobody has configured.
		return
	}

	rule := rules[0]

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

	s.logOverrideAction(created, permission.OpCreate, actor, nil, created,
		"Carrier override created")

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

	s.logOverrideAction(updated, permission.OpUpdate, actor, previous, updated,
		"Carrier override updated")

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

	s.logOverrideAction(previous, permission.OpDelete, actor, previous, nil,
		"Carrier override removed; this jurisdiction reverts to its statutory limits")

	return nil
}

// logOverrideAction records an override change against the tenant that owns it.
//
// Unlike a jurisdiction rule, an override is tenant data, so the entry carries
// the organization and business unit. It is Critical for the same reason the
// rule entries are: the row decides whether a load is flagged before it leaves.
func (s *service) logOverrideAction(
	entity *jurisdictionrule.Override,
	op permission.Operation,
	actor *services.RequestActor,
	previous any,
	current any,
	comment string,
) {
	if s.auditService == nil || entity == nil {
		return
	}

	auditActor := actor.AuditActorOrSystem()

	params := &services.LogActionParams{
		Resource:       permission.ResourceJurisdictionRuleOverride,
		ResourceID:     entity.ID.String(),
		Operation:      op,
		UserID:         auditActor.UserID,
		APIKeyID:       auditActor.APIKeyID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Critical:       true,
	}
	if current != nil {
		params.CurrentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	opts := []services.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithMetadata(map[string]any{
			"stateId": entity.StateID.String(),
			"reason":  entity.Reason,
		}),
	}
	if previous != nil && current != nil {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log jurisdiction override action",
			zap.String("overrideId", entity.ID.String()), zap.Error(err))
	}
}
