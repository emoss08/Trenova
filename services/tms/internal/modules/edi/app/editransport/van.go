package editransport

import (
	"context"
	"path"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/shared/maputils"
)

type VANTransport struct {
	network sftpNetwork
}

func NewVANTransport(cfg *config.Config) *VANTransport {
	return &VANTransport{network: sftpNetworkFromConfig(cfg)}
}

func (t *VANTransport) Method() edi.ConnectionMethod {
	return edi.ConnectionMethodVAN
}

func (t *VANTransport) Deliver(
	ctx context.Context,
	req *editransportports.EDITransportRequest,
) (*editransportports.EDITransportResult, error) {
	if req == nil || req.Profile == nil {
		return nil, ErrEDICommunicationProfileRequired
	}

	mailboxID := maputils.StringValue(req.Profile.Config, configKeyVANMailboxID)
	if mailboxID == "" {
		return nil, ErrVanMailboxIDRequired
	}

	return deliverOverSFTP(ctx, t.network, req, path.Join("/", mailboxID, "outbound"))
}
