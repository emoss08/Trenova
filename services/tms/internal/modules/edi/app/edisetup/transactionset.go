package edisetup

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) SelectTransactionSetOptions(
	ctx context.Context,
	req *edipartnerports.EDITransactionSetSelectOptionsRequest,
) (*pagination.ListResult[*edidocument.EDITransactionSet], error) {
	return s.transactionSetRepo.SelectTransactionSetOptions(ctx, req)
}
