package servicefailureservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *service) PreviewResolve(
	ctx context.Context,
	req *services.ServiceFailureLifecycleRequest,
	actor *services.RequestActor,
) (*services.ServiceFailureLifecyclePreview, error) {
	plan, err := s.planLifecycle(ctx, lifecycleTransitionParams{
		req:     req,
		actor:   actor,
		next:    servicefailure.StatusResolved,
		comment: "Service failure resolved",
	})
	if err != nil {
		return nil, err
	}

	return &services.ServiceFailureLifecyclePreview{
		Before: plan.original,
		After:  plan.updated,
		EDI:    plan.edi,
	}, nil
}

type lifecyclePlan struct {
	original *servicefailure.ServiceFailure
	updated  *servicefailure.ServiceFailure
	edi      *services.ServiceFailure214LifecycleResult
}

// planLifecycle is everything a lifecycle change decides before it saves:
// the failure may still change, the reason code is active, the status moves
// with who moved it and why, and a mandatory EDI 214 would not be blocked.
func (s *service) planLifecycle(
	ctx context.Context,
	params lifecycleTransitionParams,
) (*lifecyclePlan, error) {
	if multiErr := params.req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	actorUserID := params.actor.UserIDOrNil()
	if actorUserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Service failure lifecycle actions require a user actor",
		)
	}

	original, err := s.repo.GetByShipment(ctx, &repositories.GetServiceFailureByShipmentRequest{
		ID:         params.req.ID,
		ShipmentID: params.req.ShipmentID,
		TenantInfo: params.req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.IsTerminal() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Terminal service failures cannot be changed",
		)
	}
	if params.next == servicefailure.StatusReviewed &&
		original.Status != servicefailure.StatusOpen {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only open service failures can be reviewed",
		)
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.Version = params.req.Version
	updated.Status = params.next
	if params.req.ReasonCodeID.IsNotNil() {
		reason, reasonErr := s.activeReasonCode(ctx, activeReasonCodeParams{
			reasonCodeID: params.req.ReasonCodeID,
			tenantInfo:   params.req.TenantInfo,
			stop:         original.Stop,
		})
		if reasonErr != nil {
			return nil, reasonErr
		}
		updated.ReasonCodeID = pulid.PtrOrNil(reason.ID)
		updated.ReasonCode = reason
	}
	switch params.next {
	case servicefailure.StatusReviewed:
		if updated.ReasonCodeID == nil || updated.ReasonCodeID.IsNil() {
			return nil, reasonRequiredError()
		}
		updated.ReviewedAt = &now
		updated.ReviewedByID = pulid.PtrOrNil(actorUserID)
		if strings.TrimSpace(params.req.Notes) != "" {
			updated.InternalNotes = strings.TrimSpace(params.req.Notes)
		}
	case servicefailure.StatusResolved:
		if updated.ReasonCodeID == nil || updated.ReasonCodeID.IsNil() {
			return nil, reasonRequiredError()
		}
		updated.ResolvedAt = &now
		updated.ResolvedByID = pulid.PtrOrNil(actorUserID)
		if strings.TrimSpace(params.req.Notes) != "" {
			updated.InternalNotes = strings.TrimSpace(params.req.Notes)
		}
	case servicefailure.StatusVoided:
		if strings.TrimSpace(params.req.Notes) == "" {
			return nil, errortypes.NewValidationError(
				"notes",
				errortypes.ErrRequired,
				"Void reason is required",
			)
		}
		updated.VoidedAt = &now
		updated.VoidedByID = pulid.PtrOrNil(actorUserID)
		updated.VoidReason = strings.TrimSpace(params.req.Notes)
	case servicefailure.StatusOpen:
	}

	if multiErr := validateServiceFailure(&updated); multiErr != nil {
		return nil, multiErr
	}
	edi, err := s.preflightServiceFailure214(ctx, serviceFailure214Params{
		previous: original,
		current:  &updated,
		actor:    params.actor,
	})
	if err != nil {
		return nil, err
	}

	return &lifecyclePlan{original: original, updated: &updated, edi: edi}, nil
}
