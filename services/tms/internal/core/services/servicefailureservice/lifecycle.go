package servicefailureservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type lifecycleTransitionParams struct {
	req     *services.ServiceFailureLifecycleRequest
	actor   *services.RequestActor
	next    servicefailure.Status
	comment string
}

func (s *service) Update(
	ctx context.Context,
	req *services.UpdateServiceFailureRequest,
	actor *services.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByShipment(ctx, &repositories.GetServiceFailureByShipmentRequest{
		ID:         req.ID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.IsTerminal() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Terminal service failures cannot be updated",
		)
	}

	updated := *original
	updated.Version = req.Version
	updated.Notes = strings.TrimSpace(req.Notes)
	updated.InternalNotes = strings.TrimSpace(req.InternalNotes)
	updated.X12StatusCodeOverride = strings.TrimSpace(req.X12StatusCodeOverride)
	updated.X12ReasonCodeOverride = strings.TrimSpace(req.X12ReasonCodeOverride)
	updated.X12ExceptionCode = strings.TrimSpace(req.X12ExceptionCode)
	switch {
	case req.ClearReasonCode:
		if original.Status == servicefailure.StatusReviewed {
			return nil, errortypes.NewValidationError(
				"reasonCodeId",
				errortypes.ErrInvalidOperation,
				"Reviewed service failures must retain a reason code",
			)
		}
		updated.ReasonCodeID = nil
	case req.ReasonCodeID.IsNotNil():
		reason, reasonErr := s.activeReasonCode(ctx, activeReasonCodeParams{
			reasonCodeID: req.ReasonCodeID,
			tenantInfo:   req.TenantInfo,
			stop:         original.Stop,
		})
		if reasonErr != nil {
			return nil, reasonErr
		}
		updated.ReasonCodeID = pulid.PtrOrNil(reason.ID)
	}

	if multiErr := validateServiceFailure(&updated); multiErr != nil {
		return nil, multiErr
	}

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		return nil, err
	}

	s.logServiceFailureAction(serviceFailureActionParams{
		entity:   saved,
		actor:    actor,
		op:       permission.OpUpdate,
		previous: original,
		current:  saved,
		comment:  "Service failure updated",
	})
	s.publishInvalidation(ctx, serviceFailureInvalidationParams{
		entity:  saved,
		actor:   actor,
		action:  "updated",
		payload: saved,
	})
	if reasonChanged(original, saved) {
		s.comment(ctx, commentParams{
			entity:   saved,
			comment:  "Service failure reason changed",
			metadata: serviceFailureLifecycleMetadata(original, saved, actor),
		})
	}
	return saved, nil
}

func (s *service) Review(
	ctx context.Context,
	req *services.ServiceFailureLifecycleRequest,
	actor *services.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	return s.lifecycle(ctx, lifecycleTransitionParams{
		req:     req,
		actor:   actor,
		next:    servicefailure.StatusReviewed,
		comment: "Service failure reviewed",
	})
}

func (s *service) Resolve(
	ctx context.Context,
	req *services.ServiceFailureLifecycleRequest,
	actor *services.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	return s.lifecycle(ctx, lifecycleTransitionParams{
		req:     req,
		actor:   actor,
		next:    servicefailure.StatusResolved,
		comment: "Service failure resolved",
	})
}

func (s *service) Void(
	ctx context.Context,
	req *services.ServiceFailureLifecycleRequest,
	actor *services.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	return s.lifecycle(ctx, lifecycleTransitionParams{
		req:     req,
		actor:   actor,
		next:    servicefailure.StatusVoided,
		comment: "Service failure voided",
	})
}

func (s *service) lifecycle(
	ctx context.Context,
	params lifecycleTransitionParams,
) (*servicefailure.ServiceFailure, error) {
	if multiErr := params.req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	actorUserID := params.actor.UserIDOrNil()
	if actorUserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Service failure lifecycle actions require a user actor")
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
	if params.next == servicefailure.StatusReviewed && original.Status != servicefailure.StatusOpen {
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

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		return nil, err
	}

	op := permission.OpUpdate
	if params.next == servicefailure.StatusReviewed {
		op = permission.OpApprove
	}
	if params.next == servicefailure.StatusVoided {
		op = permission.OpArchive
	}
	s.logServiceFailureAction(serviceFailureActionParams{
		entity:   saved,
		actor:    params.actor,
		op:       op,
		previous: original,
		current:  saved,
		comment:  params.comment,
	})
	s.publishInvalidation(ctx, serviceFailureInvalidationParams{
		entity:  saved,
		actor:   params.actor,
		action:  strings.ToLower(strings.TrimPrefix(string(params.next), "Status")),
		payload: saved,
	})
	s.comment(ctx, commentParams{
		entity:   saved,
		comment:  params.comment,
		metadata: serviceFailureLifecycleMetadata(original, saved, params.actor),
	})
	return saved, nil
}

func reasonRequiredError() error {
	return errortypes.NewValidationError(
		"reasonCodeId",
		errortypes.ErrRequired,
		"Reason code is required",
	)
}

func reasonChanged(previous, current *servicefailure.ServiceFailure) bool {
	switch {
	case previous == nil || current == nil:
		return false
	case previous.ReasonCodeID == nil && current.ReasonCodeID == nil:
		return false
	case previous.ReasonCodeID == nil || current.ReasonCodeID == nil:
		return true
	default:
		return *previous.ReasonCodeID != *current.ReasonCodeID
	}
}
