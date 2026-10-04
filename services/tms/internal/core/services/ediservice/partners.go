package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListPartners(
	ctx context.Context,
	req *repositories.ListEDIPartnersRequest,
) (*pagination.ListResult[*edi.EDIPartner], error) {
	return s.partnerRepo.List(ctx, req)
}

func (s *Service) SelectPartnerOptions(
	ctx context.Context,
	req *repositories.EDIPartnerSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIPartner], error) {
	return s.partnerRepo.SelectOptions(ctx, req)
}

func (s *Service) GetPartner(
	ctx context.Context,
	req repositories.GetEDIPartnerByIDRequest,
) (*edi.EDIPartner, error) {
	return s.partnerRepo.GetByID(ctx, req)
}

func (s *Service) CreatePartner(
	ctx context.Context,
	entity *edi.EDIPartner,
	actor *services.RequestActor,
) (*edi.EDIPartner, error) {
	if err := s.requireIntegrations(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}); err != nil {
		return nil, err
	}
	normalizePartnerForCreate(entity)
	if multiErr := s.validator.ValidatePartner(entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.partnerRepo.Create(ctx, entity)
	if err != nil {
		return nil, mapEDIPartnerConstraint(err)
	}

	s.logAction(created, actor, permission.OpCreate, nil, created, "EDI partner created")
	return created, nil
}

func (s *Service) UpdatePartner(
	ctx context.Context,
	entity *edi.EDIPartner,
	actor *services.RequestActor,
) (*edi.EDIPartner, error) {
	if multiErr := s.validator.ValidatePartner(entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.partnerRepo.Update(ctx, entity)
	if err != nil {
		return nil, mapEDIPartnerConstraint(err)
	}

	s.logAction(updated, actor, permission.OpUpdate, original, updated, "EDI partner updated")
	return updated, nil
}

func (s *Service) CreateInternalPartnerPair(
	ctx context.Context,
	req *CreateInternalPartnerPairRequest,
	actor *services.RequestActor,
) (*edi.InternalPartnerPair, error) {
	if req == nil || req.TargetOrganizationID.IsNil() {
		return nil, errortypes.NewValidationError(
			"targetOrganizationId",
			errortypes.ErrRequired,
			"Target organization is required",
		)
	}
	if req.TargetOrganizationID == req.TenantInfo.OrgID {
		return nil, errortypes.NewValidationError(
			"targetOrganizationId",
			errortypes.ErrInvalid,
			"Target organization must be different from the current organization",
		)
	}

	return s.CreateInternalPartnerPairViaConnection(ctx, req, actor)
}

func mapEDIPartnerConstraint(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	multiErr := errortypes.NewMultiError()
	switch dberror.ExtractConstraintName(err) {
	case "idx_edi_partners_code_org":
		multiErr.Add("code", errortypes.ErrDuplicate, "EDI partner with this code already exists")
	case "idx_edi_partners_name_org":
		multiErr.Add("name", errortypes.ErrDuplicate, "EDI partner with this name already exists")
	case "idx_edi_partners_internal_relationship_org_bu":
		multiErr.Add(
			"internalOrganizationId",
			errortypes.ErrDuplicate,
			"An internal EDI partner already exists for this target organization",
		)
	default:
		return err
	}

	return multiErr
}

func normalizePartnerForCreate(entity *edi.EDIPartner) {
	if entity == nil {
		return
	}
	if entity.Kind == "" {
		entity.Kind = edi.PartnerKindExternal
	}
	if entity.Status == "" {
		entity.Status = domaintypes.StatusActive
	}
	if entity.Country == "" {
		entity.Country = "US"
	}
	if entity.Settings == nil {
		entity.Settings = map[string]any{}
	}
}
