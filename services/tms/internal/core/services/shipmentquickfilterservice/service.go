package shipmentquickfilterservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type CostProfileResolver interface {
	ResolveCostProfile(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		asOf time.Time,
	) (*costingservice.ResolvedCostProfile, error)
}

type Params struct {
	fx.In

	ShipmentControlRepo repositories.ShipmentControlRepository
	CostingService      *costingservice.Service
}

type Service struct {
	controls repositories.ShipmentControlRepository
	costs    CostProfileResolver
	now      func() time.Time
}

var _ services.ShipmentQuickFilterBasisResolver = (*Service)(nil)

func New(p Params) services.ShipmentQuickFilterBasisResolver {
	return NewWithDependencies(p.ShipmentControlRepo, p.CostingService, time.Now)
}

func NewWithDependencies(
	controls repositories.ShipmentControlRepository,
	costs CostProfileResolver,
	now func() time.Time,
) *Service {
	return &Service{controls: controls, costs: costs, now: now}
}

func (s *Service) Prepare(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	opts *repositories.ShipmentOptions,
) error {
	if opts == nil || !opts.HasQuickFilters() {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	shipment.ValidateQuickFilters(opts.QuickFilters, multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	margin, detention := shipment.QuickFiltersNeed(opts.QuickFilters)
	basis, err := s.Resolve(ctx, &services.ResolveShipmentQuickFilterBasisRequest{
		TenantInfo: tenantInfo,
		Timezone:   opts.Timezone,
		Margin:     margin,
		Detention:  detention,
	})
	if err != nil {
		return err
	}

	opts.QuickFilterBasis = basis
	return nil
}

func (s *Service) Resolve(
	ctx context.Context,
	req *services.ResolveShipmentQuickFilterBasisRequest,
) (*repositories.ShipmentQuickFilterBasis, error) {
	if !timeutils.IsValidLocation(req.Timezone) {
		return nil, errortypes.NewValidationError(
			"timezone",
			errortypes.ErrInvalid,
			"A valid IANA time zone is required",
		)
	}

	loc, err := time.LoadLocation(req.Timezone)
	if err != nil {
		return nil, err
	}

	now := s.now()
	basis := &repositories.ShipmentQuickFilterBasis{Now: now.In(loc), Location: loc}

	if req.Margin {
		profile, pErr := s.costs.ResolveCostProfile(ctx, req.TenantInfo, now)
		if pErr != nil {
			return nil, pErr
		}
		basis.Margin = &repositories.ShipmentMarginBasis{
			CostPerMile:          profile.TotalCPM,
			IncludeDeadheadMiles: profile.IncludeDeadheadMiles,
			TargetMarginPercent: costingcontrol.EffectiveTargetMarginPercent(
				profile.TargetMarginPercent,
			),
		}
	}

	if req.Detention {
		control, cErr := s.controls.Get(ctx, repositories.GetShipmentControlRequest{
			TenantInfo: req.TenantInfo,
		})
		if cErr != nil {
			return nil, cErr
		}
		basis.Detention = DetentionBasisOf(
			control.UseDetentionPolicyEngine,
			control.DetentionThreshold,
		)
	}

	return basis, nil
}

func DetentionBasisOf(usePolicyEngine bool, threshold *int16) *repositories.ShipmentDetentionBasis {
	minutes := shipmentstate.DefaultDelayThresholdMinutes
	if threshold != nil {
		minutes = shipmentstate.ResolveDelayThresholdMinutes(*threshold)
	}

	return &repositories.ShipmentDetentionBasis{
		UsePolicyEngine:  usePolicyEngine,
		ThresholdMinutes: int64(minutes),
	}
}
