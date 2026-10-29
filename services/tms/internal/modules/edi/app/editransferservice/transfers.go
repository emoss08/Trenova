package editransferservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListInboundTransfers(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
) (*pagination.ListResult[*editransfer.EDITransfer], error) {
	return s.transferRepo.ListInbound(ctx, req)
}

func (s *Service) ListOutboundTransfers(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
) (*pagination.ListResult[*editransfer.EDITransfer], error) {
	return s.transferRepo.ListOutbound(ctx, req)
}

func (s *Service) GetTransfer(
	ctx context.Context,
	req editenderports.GetEDITransferByIDRequest,
) (*editransfer.EDITransfer, error) {
	return s.transferRepo.GetTransferByID(ctx, req)
}

func (s *Service) GetTransfersByIDs(
	ctx context.Context,
	req editenderports.GetEDITransfersByIDsRequest,
) ([]*editransfer.EDITransfer, error) {
	return s.transferRepo.GetTransfersByIDs(ctx, req)
}

func (s *Service) TransferSelectOptions(
	ctx context.Context,
	req *editenderports.EDITransferSelectOptionsRequest,
) (*pagination.ListResult[*editransfer.EDITransfer], error) {
	return s.transferRepo.SelectOptions(ctx, req)
}
