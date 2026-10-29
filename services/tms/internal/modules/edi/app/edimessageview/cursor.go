package edimessageview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListMessagesCursor(
	ctx context.Context,
	req *edimessageports.ListEDIMessagesRequest,
) (*pagination.CursorListResult[*edimessage.EDIMessage], error) {
	return s.messageRepo.ListMessagesCursor(ctx, req)
}
