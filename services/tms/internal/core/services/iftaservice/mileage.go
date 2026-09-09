package iftaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DeleteMileageEntryRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	UserID     pulid.ID
}

func entryTenant(entity *ifta.JurisdictionMileageEntry) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func (s *Service) ListMileageEntries(
	ctx context.Context,
	req *repositories.ListMileageEntriesRequest,
) (*pagination.CursorListResult[*ifta.JurisdictionMileageEntry], error) {
	return s.repo.ListMileageEntries(ctx, req)
}

func (s *Service) GetMileageEntry(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*ifta.JurisdictionMileageEntry, error) {
	return s.repo.GetMileageEntryByID(ctx, &repositories.GetMileageEntryByIDRequest{
		ID:                  id,
		TenantInfo:          tenantInfo,
		IncludeTractor:      true,
		IncludeJurisdiction: true,
	})
}

func (s *Service) prepareMileageEntry(
	ctx context.Context,
	entity *ifta.JurisdictionMileageEntry,
) error {
	tenantInfo := entryTenant(entity)
	multiErr := errortypes.NewMultiError()

	if entity.TractorID.IsNil() {
		multiErr.Add("tractorId", errortypes.ErrRequired, "Tractor is required")
	} else if _, err := s.tractorRepo.GetByID(ctx, repositories.GetTractorByIDRequest{
		ID:         entity.TractorID,
		TenantInfo: tenantInfo,
	}); err != nil {
		multiErr.Add("tractorId", errortypes.ErrInvalid, "Tractor was not found in your organization")
	}

	if entity.JurisdictionID.IsNil() {
		multiErr.Add("jurisdictionId", errortypes.ErrRequired, "Jurisdiction is required")
	} else {
		jurisdiction, err := s.repo.GetJurisdictionByID(ctx, entity.JurisdictionID)
		switch {
		case err != nil:
			multiErr.Add("jurisdictionId", errortypes.ErrInvalid, "Jurisdiction was not found")
		case !jurisdiction.IsActive():
			multiErr.Add(
				"jurisdictionId",
				errortypes.ErrInvalid,
				jurisdiction.Label()+" is inactive and cannot receive mileage",
			)
		}
	}

	entity.Normalize()
	if entity.TraveledAt > 0 {
		entity.AssignPeriod(s.tenantLocation(ctx, entity.OrganizationID))
	}
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) CreateMileageEntry(
	ctx context.Context,
	entity *ifta.JurisdictionMileageEntry,
	userID pulid.ID,
) (*ifta.JurisdictionMileageEntry, error) {
	tenantInfo := entryTenant(entity)
	entity.CreatedByID = userID
	if entity.Source == "" {
		entity.Source = ifta.MileageSourceManual
	}
	if err := s.prepareMileageEntry(ctx, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateMileageEntry(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceIFTAJurisdictionMileage,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment: "Recorded " + created.Miles.StringFixed(milesScale) + " miles for " +
			created.Period().Label(),
	})
	s.publish(ctx, tenantInfo, realtimeMileageEntry, permission.OpCreate, created.ID, userID)

	return created, nil
}

func (s *Service) UpdateMileageEntry(
	ctx context.Context,
	entity *ifta.JurisdictionMileageEntry,
	userID pulid.ID,
) (*ifta.JurisdictionMileageEntry, error) {
	tenantInfo := entryTenant(entity)
	existing, err := s.repo.GetMileageEntryByID(ctx, &repositories.GetMileageEntryByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if existing.Version != entity.Version {
		return nil, versionMismatch("mileage entry")
	}

	previous := *existing
	entity.CreatedByID = existing.CreatedByID
	entity.CreatedAt = existing.CreatedAt
	if entity.Source == "" {
		entity.Source = existing.Source
	}
	if err = s.prepareMileageEntry(ctx, entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateMileageEntry(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceIFTAJurisdictionMileage,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Updated a jurisdiction mileage entry for " + updated.Period().Label(),
	})
	s.publish(ctx, tenantInfo, realtimeMileageEntry, permission.OpUpdate, updated.ID, userID)

	return updated, nil
}

func (s *Service) DeleteMileageEntry(ctx context.Context, req *DeleteMileageEntryRequest) error {
	existing, err := s.repo.GetMileageEntryByID(ctx, &repositories.GetMileageEntryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if existing.Version != req.Version {
		return versionMismatch("mileage entry")
	}

	if err = s.repo.DeleteMileageEntry(ctx, &repositories.DeleteMileageEntryRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		Version:    req.Version,
	}); err != nil {
		return err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceIFTAJurisdictionMileage,
		resourceID: existing.ID.String(),
		operation:  permission.OpDelete,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    existing,
		comment: "Deleted " + existing.Miles.StringFixed(milesScale) + " miles from " +
			existing.Period().Label(),
	})
	s.publish(ctx, req.TenantInfo, realtimeMileageEntry, permission.OpDelete, existing.ID, req.UserID)

	return nil
}
