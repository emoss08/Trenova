package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) SelectTransactionSetOptions(
	ctx context.Context,
	req *ediports.EDITransactionSetSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITransactionSet], error) {
	return s.transactionSetRepo.SelectTransactionSetOptions(ctx, req)
}
