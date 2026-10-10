package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListInboundTransfers(
	ctx context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	return s.transferRepo.ListInbound(ctx, req)
}

func (s *Service) ListOutboundTransfers(
	ctx context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	return s.transferRepo.ListOutbound(ctx, req)
}

func (s *Service) GetTransfer(
	ctx context.Context,
	req ediports.GetEDITransferByIDRequest,
) (*edi.EDITransfer, error) {
	return s.transferRepo.GetTransferByID(ctx, req)
}

func (s *Service) GetTransfersByIDs(
	ctx context.Context,
	req ediports.GetEDITransfersByIDsRequest,
) ([]*edi.EDITransfer, error) {
	return s.transferRepo.GetTransfersByIDs(ctx, req)
}

func (s *Service) TransferSelectOptions(
	ctx context.Context,
	req *ediports.EDITransferSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	return s.transferRepo.SelectOptions(ctx, req)
}
