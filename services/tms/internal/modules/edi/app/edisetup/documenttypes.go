package edisetup

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListDocumentTypes(
	ctx context.Context,
	req edipartnerports.ListEDIDocumentTypesRequest,
) ([]*edidocument.EDIDocumentType, error) {
	return s.documentTypeRepo.ListDocumentTypes(ctx, req)
}

func (s *Service) SelectDocumentTypeOptions(
	ctx context.Context,
	req *edipartnerports.EDIDocumentTypeSelectOptionsRequest,
) (*pagination.ListResult[*edidocument.EDIDocumentType], error) {
	return s.documentTypeRepo.SelectDocumentTypeOptions(ctx, req)
}
