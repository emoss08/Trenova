package edisetup

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListPartnersCursor(
	ctx context.Context,
	req *edipartnerports.ListEDIPartnersRequest,
) (*pagination.CursorListResult[*edi.EDIPartner], error) {
	return s.partnerRepo.ListCursor(ctx, req)
}

func (s *Service) ListCommunicationProfilesCursor(
	ctx context.Context,
	req *edipartnerports.ListEDICommunicationProfilesRequest,
) (*pagination.CursorListResult[*edi.EDICommunicationProfile], error) {
	return s.profileRepo.ListProfilesCursor(ctx, req)
}

func (s *Service) ListMappingProfilesCursor(
	ctx context.Context,
	req *edipartnerports.ListEDIMappingProfilesRequest,
) (*pagination.CursorListResult[*edi.EDIMappingProfile], error) {
	return s.mappingProfileRepo.ListMappingProfilesCursor(ctx, req)
}
