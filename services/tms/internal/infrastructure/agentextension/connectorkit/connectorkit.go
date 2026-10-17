package connectorkit

import (
	"errors"
	"net/http"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/shared/restx"
	"go.uber.org/fx"
)

const ConnectorGroup = `group:"agent_extension_connectors"`

func Provide(constructor any) fx.Option {
	return fx.Provide(fx.Annotate(
		constructor,
		fx.As(new(agentextensionports.ExtensionConnector)),
		fx.ResultTags(ConnectorGroup),
	))
}

func ClassifyHTTP(err error) agentextensionports.ProviderFailure {
	switch {
	case err == nil:
		return agentextensionports.ProviderFailure{}
	case restx.IsStatus(err, http.StatusUnauthorized), restx.IsStatus(err, http.StatusForbidden):
		return agentextensionports.ProviderFailure{
			Kind: agentextensionports.ProviderFailureUnauthorized,
		}
	case restx.IsStatus(err, http.StatusPaymentRequired):
		return agentextensionports.ProviderFailure{
			Kind: agentextensionports.ProviderFailureOutOfCredits,
		}
	case restx.IsRateLimited(err):
		return agentextensionports.ProviderFailure{
			Kind: agentextensionports.ProviderFailureRateLimited,
		}
	case restx.IsStatus(err, http.StatusNotFound):
		return agentextensionports.ProviderFailure{
			Kind: agentextensionports.ProviderFailureNotFound,
		}
	case restx.IsStatus(err, http.StatusBadRequest),
		restx.IsStatus(err, http.StatusUnprocessableEntity):
		return InvalidRequest(err)
	default:
		return agentextensionports.ProviderFailure{}
	}
}

func InvalidRequest(err error) agentextensionports.ProviderFailure {
	failure := agentextensionports.ProviderFailure{
		Kind: agentextensionports.ProviderFailureInvalidRequest,
	}
	var apiErr *restx.APIError
	if errors.As(err, &apiErr) {
		failure.Detail = apiErr.Message
	}

	return failure
}
