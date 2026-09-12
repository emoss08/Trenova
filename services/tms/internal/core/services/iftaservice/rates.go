package iftaservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const ratesField = "rates"

type UpsertTaxRatesRequest struct {
	TenantInfo pagination.TenantInfo
	Rates      []*ifta.TaxRate
	UserID     pulid.ID
}

type DeleteTaxRateRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	UserID     pulid.ID
}

func (s *Service) ListTaxRates(
	ctx context.Context,
	req *repositories.ListTaxRatesRequest,
) (*pagination.CursorListResult[*ifta.TaxRate], error) {
	return s.repo.ListTaxRates(ctx, req)
}

func (s *Service) GetTaxRate(ctx context.Context, id pulid.ID) (*ifta.TaxRate, error) {
	return s.repo.GetTaxRateByID(ctx, id)
}

func (s *Service) validateRates(
	ctx context.Context,
	rates []*ifta.TaxRate,
) error {
	multiErr := errortypes.NewMultiError()
	seen := make(map[ifta.RateKey]int, len(rates))
	jurisdictionIDs := make([]pulid.ID, 0, len(rates))
	seenJurisdiction := make(map[pulid.ID]struct{}, len(rates))

	for i, rate := range rates {
		scoped := multiErr.WithIndex(ratesField, i)
		if rate == nil {
			scoped.Add("", errortypes.ErrRequired, "Rate is required")
			continue
		}
		rate.Normalize()
		rate.Validate(scoped)

		key := ifta.RateKey{JurisdictionID: rate.JurisdictionID, FuelType: rate.FuelType}
		if first, dup := seen[key]; dup {
			scoped.Add(
				"fuelType",
				errortypes.ErrDuplicate,
				"This jurisdiction and fuel type already appears at row {0}; each pair may be sent once per period", strconv.Itoa(first+1),
			)
		} else {
			seen[key] = i
		}
		if !rate.JurisdictionID.IsNil() {
			if _, ok := seenJurisdiction[rate.JurisdictionID]; !ok {
				seenJurisdiction[rate.JurisdictionID] = struct{}{}
				jurisdictionIDs = append(jurisdictionIDs, rate.JurisdictionID)
			}
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	known, err := s.jurisdictionsByID(ctx, jurisdictionIDs)
	if err != nil {
		return err
	}
	for i, rate := range rates {
		if _, ok := known[rate.JurisdictionID]; !ok {
			multiErr.WithIndex(ratesField, i).Add(
				"jurisdictionId",
				errortypes.ErrInvalid,
				"Jurisdiction does not exist",
			)
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) UpsertTaxRates(
	ctx context.Context,
	req *UpsertTaxRatesRequest,
) ([]*ifta.TaxRate, error) {
	if len(req.Rates) == 0 {
		return nil, errortypes.NewValidationError(
			ratesField,
			errortypes.ErrRequired,
			"At least one rate is required",
		)
	}
	if err := s.validateRates(ctx, req.Rates); err != nil {
		return nil, err
	}

	saved, err := s.repo.UpsertTaxRates(ctx, req.Rates)
	if err != nil {
		return nil, err
	}

	for _, rate := range saved {
		s.audit(&auditParams{
			resource:   permission.ResourceIFTATaxRate,
			resourceID: rate.ID.String(),
			operation:  permission.OpManage,
			userID:     req.UserID,
			tenant:     req.TenantInfo,
			current:    rate,
			comment: "Set the " + rate.FuelType.Label() + " rate for " +
				rate.Period().Label(),
		})
		s.publish(ctx, req.TenantInfo, realtimeTaxRate, permission.OpManage, rate.ID, req.UserID)
	}

	return saved, nil
}

func (s *Service) DeleteTaxRate(ctx context.Context, req *DeleteTaxRateRequest) error {
	existing, err := s.repo.GetTaxRateByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if existing.Version != req.Version {
		return versionMismatch("tax rate")
	}

	if err = s.repo.DeleteTaxRate(ctx, req.ID, req.Version); err != nil {
		return err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceIFTATaxRate,
		resourceID: existing.ID.String(),
		operation:  permission.OpDelete,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    existing,
		comment: "Removed the " + existing.FuelType.Label() + " rate for " +
			existing.Period().Label(),
	})
	s.publish(ctx, req.TenantInfo, realtimeTaxRate, permission.OpDelete, existing.ID, req.UserID)

	return nil
}
