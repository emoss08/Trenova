package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) GetMappingProfile(
	ctx context.Context,
	req repositories.GetMappingProfileRequest,
) (*edi.EDIMappingProfile, error) {
	return s.mappingProfileRepo.GetMappingProfile(ctx, req)
}

func (s *Service) ListMappingProfiles(
	ctx context.Context,
	req *repositories.ListEDIMappingProfilesRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return s.mappingProfileRepo.ListMappingProfiles(ctx, req)
}

func (s *Service) SelectMappingProfileOptions(
	ctx context.Context,
	req *repositories.EDIMappingProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return s.mappingProfileRepo.SelectMappingProfileOptions(ctx, req)
}

func (s *Service) GetMappingProfileByID(
	ctx context.Context,
	req repositories.GetMappingProfileByIDRequest,
) (*edi.EDIMappingProfile, error) {
	return s.mappingProfileRepo.GetMappingProfileByID(ctx, req)
}

func (s *Service) SaveMappingProfile(
	ctx context.Context,
	req *repositories.SaveMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	if multiErr := s.validator.ValidateMappingItems(req.Items); multiErr != nil {
		return nil, multiErr
	}

	return s.mappingProfileRepo.SaveMappingItems(ctx, req)
}

func (s *Service) SaveMappingProfileItems(
	ctx context.Context,
	req *repositories.SaveMappingProfileItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	if multiErr := s.validator.ValidateMappingItems(req.Items); multiErr != nil {
		return nil, multiErr
	}

	return s.mappingProfileRepo.SaveMappingProfileItems(ctx, req)
}

func (s *Service) DeleteMappingItem(
	ctx context.Context,
	req repositories.DeleteMappingItemRequest,
) error {
	return s.mappingProfileRepo.DeleteMappingItem(ctx, req)
}

func (s *Service) DeleteMappingProfileItem(
	ctx context.Context,
	req repositories.DeleteMappingProfileItemRequest,
) error {
	return s.mappingProfileRepo.DeleteMappingProfileItem(ctx, req)
}
