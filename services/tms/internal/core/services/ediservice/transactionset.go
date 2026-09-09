package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) SelectTransactionSetOptions(
	ctx context.Context,
	req *repositories.EDITransactionSetSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITransactionSet], error) {
	return s.transactionSetRepo.SelectTransactionSetOptions(ctx, req)
}
