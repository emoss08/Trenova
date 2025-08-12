package partnerconfig

import (
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/grpcutils"
	configpb "github.com/emoss08/trenova/shared/edi/proto/config/v1"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PartnerClientParams struct {
	fx.In

	LC     fx.Lifecycle
	Config *config.Config
	Logger *logger.Logger
}

type PartnerConfigClient struct {
	l   *zerolog.Logger
	c   configpb.EDIConfigServiceClient
	cfg config.GRPCServerConfig
}

func NewPartnerConfigClient(p PartnerClientParams) (configpb.EDIConfigServiceClient, error) {
	addr := grpcutils.NormalizeDialTarget(p.Config.GRPC.ListenAddress)
	log := p.Logger.With().
		Str("service", "edi-partner-config-client").
		Str("address", addr).
		Logger()

	client, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to EDIConfigService")
		return nil, err
	}

	cli := configpb.NewEDIConfigServiceClient(client)

	return cli, nil
}
