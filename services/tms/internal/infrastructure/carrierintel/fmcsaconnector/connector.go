package fmcsaconnector

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/hostguard"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/intelkit"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/carrierintelligenceconfig"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelusage"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/fmcsa"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	limiterNamespace = "fmcsa"
	testDOTNumber    = "44110"

	messageWebKeyRequired = "FMCSA QCMobile web key is required"
	messageKeyRejected    = "FMCSA rejected the web key"
	messageUnreachable    = "Could not reach FMCSA QCMobile"
	messageMisconfigured  = "FMCSA QCMobile integration is not configured correctly"
)

var (
	_ carrierintelports.CarrierIntelConnector = (*Connector)(nil)

	errorMapper = intelkit.ErrorMapper{
		Provider:     integration.TypeFMCSAQCMobile,
		Unauthorized: fmcsa.IsUnauthorized,
		NotFound:     fmcsa.IsNotFound,
		RateLimited:  fmcsa.IsRateLimited,
		Invalid:      isInvalidArgument,
	}
)

type Connector struct {
	appConfig                 *appconfig.Config
	carrierIntelligenceConfig *carrierintelligenceconfig.Config
	extraOptions              []fmcsa.Option
}

func New(
	appConfig *appconfig.Config,
	carrierIntelligenceConfig *carrierintelligenceconfig.Config,
) *Connector {
	return &Connector{appConfig: appConfig,
		carrierIntelligenceConfig: carrierIntelligenceConfig}
}

func isInvalidArgument(err error) bool {
	return errors.Is(err, fmcsa.ErrInvalidArgument)
}

func (c *Connector) IntegrationType() integration.Type {
	return integration.TypeFMCSAQCMobile
}

func (c *Connector) Descriptor() carrierintel.ProviderDescriptor {
	return carrierintel.ProviderDescriptor{
		Capabilities: carrierintel.NewCapabilitySet(
			carrierintel.CapabilityLookupFMCSA,
			carrierintel.CapabilitySearch,
			carrierintel.CapabilitySnapshotMonitoring,
			carrierintel.CapabilityBrokerAuthority,
		),
		Sections: []carrierintel.Section{
			carrierintel.SectionIdentity,
			carrierintel.SectionAuthority,
			carrierintel.SectionInsurance,
			carrierintel.SectionSafety,
			carrierintel.SectionBasics,
			carrierintel.SectionInspections,
			carrierintel.SectionCrashes,
			carrierintel.SectionFleet,
			carrierintel.SectionOperations,
		},
	}
}

func (c *Connector) PriceBook() carrierintelusage.PriceBook {
	return carrierintelusage.PriceBook{}
}

func (c *Connector) TestConnection(ctx context.Context, cfg map[string]string) error {
	sdk, err := c.newSDKClient(cfg, nil)
	if err != nil {
		return err
	}

	_, err = sdk.CarrierByDOT(
		carrierintel.WithPurpose(ctx, carrierintel.PurposeTest),
		testDOTNumber,
	)
	switch {
	case err == nil, fmcsa.IsNotFound(err):
		return nil
	case fmcsa.IsUnauthorized(err):
		return intelkit.NewUserError(messageKeyRejected, err)
	default:
		return intelkit.NewUserError(messageUnreachable, err)
	}
}

func (c *Connector) Bind(
	params *carrierintelports.CarrierIntelBindParams,
) (carrierintelports.CarrierIntelClient, error) {
	if params == nil {
		return nil, intelkit.NewUserError(messageWebKeyRequired, nil)
	}
	sdk, err := c.newSDKClient(params.Config, params.Limiter)
	if err != nil {
		return nil, err
	}
	return &Client{
		sdk:      sdk,
		recorder: intelkit.NewRecorder(params.Recorder),
	}, nil
}

func (c *Connector) newSDKClient(
	cfg map[string]string,
	limiter restx.Limiter,
) (*fmcsa.Client, error) {
	webKey := strings.TrimSpace(cfg[integration.ConfigKeyCarrierIntelWebKey])
	if webKey == "" {
		return nil, intelkit.NewUserError(messageWebKeyRequired, nil)
	}

	target, err := hostguard.Resolve(
		cfg[integration.ConfigKeyCarrierIntelBaseURL],
		integration.DefaultFMCSABaseURL,
		c.appConfig, c.carrierIntelligenceConfig,
	)
	if err != nil {
		return nil, err
	}

	options := make([]fmcsa.Option, 0, 3+len(c.extraOptions))
	options = append(options,
		fmcsa.WithBaseURL(target.BaseURL),
		fmcsa.WithHTTPClient(target.HTTPClient(hostguard.DefaultTimeout)),
	)
	if limiter != nil {
		options = append(options,
			fmcsa.WithLimiter(limiter, intelkit.LimiterKeyPrefix(limiterNamespace, webKey)),
		)
	}
	options = append(options, c.extraOptions...)

	sdk, err := fmcsa.New(webKey, options...)
	if err != nil {
		return nil, intelkit.NewUserError(messageMisconfigured, err)
	}
	return sdk, nil
}
