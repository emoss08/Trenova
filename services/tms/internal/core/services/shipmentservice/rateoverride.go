package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// SetRateOverride replaces the contract's answer with a hand-set rate, or
// removes one.
//
// This is the only path that writes the override fields: every other save
// restores them from the original, so a client round-tripping a shipment
// cannot clear a rater's override as a side effect. The shipment is re-rated
// immediately, which is what records the override on a quote — including what
// the contract would have charged instead, which is the rate leakage report.
func (s *service) SetRateOverride(
	ctx context.Context,
	req *services.SetRateOverrideRequest,
	actor *services.RequestActor,
) (*shipment.Shipment, error) {
	if multiErr := validateRateOverrideRequest(req); multiErr != nil {
		return nil, multiErr
	}

	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "SetRateOverride"),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		log.Error("failed to get shipment for rate override", zap.Error(err))
		return nil, err
	}

	if multiErr := validateShipmentNotLockedForBilling(original); multiErr != nil {
		return nil, multiErr
	}

	if multiErr := s.enforceOverrideReason(ctx, req); multiErr != nil {
		return nil, multiErr
	}

	entity, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	applyOverride(entity, req, auditActor.UserID)

	control, err := s.getShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	// The recalculation runs before the lock lands, so an override being set is
	// applied — and its quote written — rather than frozen out by its own lock.
	// The lock then holds whatever the recalculation produced.
	if err = s.commercial.Recalculate(ctx, entity, control, auditActor.UserID); err != nil {
		return nil, err
	}

	entity.RateLocked = req.RateLocked

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to save rate override", zap.Error(err))
		return nil, err
	}

	comment := "Rate override set"
	if req.Clear {
		comment = "Rate override cleared"
	}

	if err = s.logShipmentAction(
		updatedEntity,
		auditActor,
		permission.OpUpdate,
		original,
		updatedEntity,
		auditservice.WithComment(comment),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	if err = s.publishShipmentInvalidation(
		ctx, updatedEntity, auditActor, "rate-override", updatedEntity,
	); err != nil {
		log.Warn("failed to publish realtime invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}

func validateRateOverrideRequest(req *services.SetRateOverrideRequest) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if req == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Rate override request is required")
		return multiErr
	}

	if !req.Clear {
		if !req.Amount.Valid {
			multiErr.Add("amount", errortypes.ErrRequired, "An override amount is required")
		} else if req.Amount.Decimal.IsNegative() {
			multiErr.Add("amount", errortypes.ErrInvalid, "An override amount cannot be negative")
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// enforceOverrideReason holds an override to the organization's own policy: a
// departure from the contract has to say why when the billing control demands
// it, because that reason is what the audit trail explains the invoice with.
func (s *service) enforceOverrideReason(
	ctx context.Context,
	req *services.SetRateOverrideRequest,
) *errortypes.MultiError {
	if req.Clear || req.Reason != "" || s.billingRepo == nil {
		return nil
	}

	billingControl, err := s.billingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		// A billing control that cannot be read is a policy that cannot be
		// enforced, not a reason to block the override — the same tolerance
		// the billing readiness checks extend it.
		return nil //nolint:nilerr // an unreadable policy is no policy
	}

	if billingControl != nil && billingControl.RequireRateOverrideReason {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"reason",
			errortypes.ErrRequired,
			"Your organization requires a reason for every rate override",
		)
		return multiErr
	}

	return nil
}

func applyOverride(
	entity *shipment.Shipment,
	req *services.SetRateOverrideRequest,
	userID pulid.ID,
) {
	if req.Clear {
		entity.RateOverrideAmount = decimal.NullDecimal{}
		entity.RateOverrideReason = ""
		entity.RateOverrideByID = nil
		entity.RateOverrideAt = nil
		entity.RateLocked = false
		return
	}

	now := timeutils.NowUnix()

	entity.RateOverrideAmount = req.Amount
	entity.RateOverrideReason = req.Reason
	entity.RateOverrideByID = &userID
	entity.RateOverrideAt = &now
	// The lock is applied after the recalculation, so the override lands first.
	entity.RateLocked = false
}
