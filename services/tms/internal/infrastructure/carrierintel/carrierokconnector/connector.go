package carrierokconnector

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/hostguard"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/intelkit"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/carrierintelligenceconfig"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelusage"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

const (
	limiterNamespace     = "carrierok"
	testAutocompleteTerm = "ac"

	messageAPIKeyRequired    = "CarrierOk API key is required"
	messageSandboxDisallowed = "Sandbox API keys are not allowed in this environment"
	messageKeyRejected       = "CarrierOk rejected the API key"
	messagePaymentFailed     = "CarrierOk reports the account's payment failed"
	messageUnreachable       = "Could not reach CarrierOk"
	messageMisconfigured     = "CarrierOk integration is not configured correctly"
)

var (
	_ carrierintelports.CarrierIntelConnector = (*Connector)(nil)

	errorMapper = intelkit.ErrorMapper{
		Provider:        integration.TypeCarrierOK,
		Unauthorized:    carrierok.IsUnauthorized,
		PaymentRequired: carrierok.IsPaymentRequired,
		NotFound:        carrierok.IsNotFound,
		RateLimited:     carrierok.IsRateLimited,
		Invalid:         isInvalidQuery,
	}
)

type Connector struct {
	appConfig                 *appconfig.Config
	carrierIntelligenceConfig *carrierintelligenceconfig.Config
	extraOptions              []carrierok.Option
}

func New(
	appConfig *appconfig.Config,
	carrierIntelligenceConfig *carrierintelligenceconfig.Config,
) *Connector {
	return &Connector{appConfig: appConfig,
		carrierIntelligenceConfig: carrierIntelligenceConfig}
}

func (c *Connector) IntegrationType() integration.Type {
	return integration.TypeCarrierOK
}

func (c *Connector) Descriptor() carrierintel.ProviderDescriptor {
	return carrierintel.ProviderDescriptor{
		Capabilities: carrierintel.NewCapabilitySet(
			carrierintel.CapabilityLookupFull,
			carrierintel.CapabilityLookupLite,
			carrierintel.CapabilityLookupFMCSA,
			carrierintel.CapabilitySearch,
			carrierintel.CapabilityAutocomplete,
			carrierintel.CapabilityNativeMonitoring,
			carrierintel.CapabilityEquipmentLookup,
			carrierintel.CapabilityNetworkSignals,
			carrierintel.CapabilityLanes,
			carrierintel.CapabilityBrokerAuthority,
			carrierintel.CapabilityInsuranceHistory,
			carrierintel.CapabilityRiskScore,
		),
		Sections: carrierintel.AllSections(),
	}
}

func (c *Connector) PriceBook() carrierintelusage.PriceBook {
	return carrierintelusage.PriceBook{
		carrierintelusage.EndpointProfileFull: price(
			carrierintelusage.BillingModelPerDOTMonth,
			"3.00",
		),
		carrierintelusage.EndpointProfileLite: price(
			carrierintelusage.BillingModelPerDOTMonth,
			"0.50",
		),
		carrierintelusage.EndpointProfileFMCSA: price(
			carrierintelusage.BillingModelPerMatch,
			"0.10",
		),
		carrierintelusage.EndpointAutocomplete: price(
			carrierintelusage.BillingModelPerRequest,
			"0.003",
		),
		carrierintelusage.EndpointSearch: price(
			carrierintelusage.BillingModelPerRequest,
			"0.003",
		),
		carrierintelusage.EndpointMonitorAdd: price(
			carrierintelusage.BillingModelPerDOTMonth,
			"0.50",
		),
		carrierintelusage.EndpointMonitorRemove: price(carrierintelusage.BillingModelFree, "0"),
		carrierintelusage.EndpointMonitorList:   price(carrierintelusage.BillingModelFree, "0"),
		carrierintelusage.EndpointEquipment: price(
			carrierintelusage.BillingModelPerDOTMonth,
			"3.00",
		),
		carrierintelusage.EndpointComposite: price(carrierintelusage.BillingModelFree, "0"),
	}
}

func price(model carrierintelusage.BillingModel, unitCost string) carrierintelusage.EndpointPrice {
	return carrierintelusage.EndpointPrice{
		Model:    model,
		UnitCost: decimal.RequireFromString(unitCost),
	}
}

func (c *Connector) TestConnection(ctx context.Context, cfg map[string]string) error {
	sdk, err := c.newSDKClient(cfg, nil)
	if err != nil {
		return err
	}

	_, err = sdk.Autocomplete(
		carrierintel.WithPurpose(ctx, carrierintel.PurposeTest),
		testAutocompleteTerm,
		1,
	)
	switch {
	case err == nil:
		return nil
	case carrierok.IsUnauthorized(err):
		return intelkit.NewUserError(messageKeyRejected, err)
	case carrierok.IsPaymentRequired(err):
		return intelkit.NewUserError(messagePaymentFailed, err)
	default:
		return intelkit.NewUserError(messageUnreachable, err)
	}
}

func (c *Connector) Bind(
	params *carrierintelports.CarrierIntelBindParams,
) (carrierintelports.CarrierIntelClient, error) {
	if params == nil {
		return nil, intelkit.NewUserError(messageAPIKeyRequired, nil)
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
) (*carrierok.Client, error) {
	apiKey := strings.TrimSpace(cfg[integration.ConfigKeyCarrierIntelAPIKey])
	if apiKey == "" {
		return nil, intelkit.NewUserError(messageAPIKeyRequired, nil)
	}
	if carrierok.IsSandboxKey(apiKey) &&
		!c.carrierIntelligenceConfig.IsSandboxAllowed(c.appConfig) {
		return nil, intelkit.NewUserError(messageSandboxDisallowed, nil)
	}

	target, err := hostguard.Resolve(
		cfg[integration.ConfigKeyCarrierIntelBaseURL],
		integration.DefaultCarrierOKBaseURL,
		c.appConfig, c.carrierIntelligenceConfig,
	)
	if err != nil {
		return nil, err
	}

	options := make([]carrierok.Option, 0, 3+len(c.extraOptions))
	options = append(options,
		carrierok.WithBaseURL(target.BaseURL),
		carrierok.WithHTTPClient(target.HTTPClient(hostguard.DefaultTimeout)),
	)
	if limiter != nil {
		options = append(options,
			carrierok.WithLimiter(limiter, intelkit.LimiterKeyPrefix(limiterNamespace, apiKey)),
		)
	}
	options = append(options, c.extraOptions...)

	sdk, err := carrierok.New(apiKey, options...)
	if err != nil {
		return nil, intelkit.NewUserError(messageMisconfigured, err)
	}
	return sdk, nil
}
