package editransport

import (
	"context"
	"path"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/shared/maputils"
)

type VANTransport struct{}

func NewVANTransport() *VANTransport {
	return &VANTransport{}
}

func (t *VANTransport) Method() edi.ConnectionMethod {
	return edi.ConnectionMethodVAN
}

func (t *VANTransport) Deliver(
	ctx context.Context,
	req *ediports.EDITransportRequest,
) (*ediports.EDITransportResult, error) {
	if req == nil || req.Profile == nil {
		return nil, ErrEDICommunicationProfileRequired
	}

	mailboxID := maputils.StringValue(req.Profile.Config, configKeyVANMailboxID)
	if mailboxID == "" {
		return nil, ErrVanMailboxIDRequired
	}

	return deliverOverSFTP(ctx, req, path.Join("/", mailboxID, "outbound"))
}
