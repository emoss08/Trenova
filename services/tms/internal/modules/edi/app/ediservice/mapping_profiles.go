package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) GetMappingProfile(
	ctx context.Context,
	req edipartnerports.GetMappingProfileRequest,
) (*edi.EDIMappingProfile, error) {
	return s.mappingProfileRepo.GetMappingProfile(ctx, req)
}

func (s *Service) ListMappingProfiles(
	ctx context.Context,
	req *edipartnerports.ListEDIMappingProfilesRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return s.mappingProfileRepo.ListMappingProfiles(ctx, req)
}

func (s *Service) SelectMappingProfileOptions(
	ctx context.Context,
	req *edipartnerports.EDIMappingProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return s.mappingProfileRepo.SelectMappingProfileOptions(ctx, req)
}

func (s *Service) GetMappingProfileByID(
	ctx context.Context,
	req edipartnerports.GetMappingProfileByIDRequest,
) (*edi.EDIMappingProfile, error) {
	return s.mappingProfileRepo.GetMappingProfileByID(ctx, req)
}

func (s *Service) SaveMappingProfile(
	ctx context.Context,
	req *edipartnerports.SaveMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	if multiErr := s.validator.ValidateMappingItems(req.Items); multiErr != nil {
		return nil, multiErr
	}

	return s.mappingProfileRepo.SaveMappingItems(ctx, req)
}

func (s *Service) SaveMappingProfileItems(
	ctx context.Context,
	req *edipartnerports.SaveMappingProfileItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	if multiErr := s.validator.ValidateMappingItems(req.Items); multiErr != nil {
		return nil, multiErr
	}

	return s.mappingProfileRepo.SaveMappingProfileItems(ctx, req)
}

func (s *Service) DeleteMappingItem(
	ctx context.Context,
	req edipartnerports.DeleteMappingItemRequest,
) error {
	return s.mappingProfileRepo.DeleteMappingItem(ctx, req)
}

func (s *Service) DeleteMappingProfileItem(
	ctx context.Context,
	req edipartnerports.DeleteMappingProfileItemRequest,
) error {
	return s.mappingProfileRepo.DeleteMappingProfileItem(ctx, req)
}
