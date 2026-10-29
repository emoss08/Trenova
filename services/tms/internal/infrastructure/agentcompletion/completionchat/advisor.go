package completionchat

import (
	"errors"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const (
	advisorField = "advisor"
)

// advisorFor is the advisor a chat call to provider declares. A caller that
// names one, such as a desk bench cell, overrides the provider's setting;
// otherwise the provider's own setting decides.
func advisorFor(
	provider *aiprovider.Provider,
	requested *llmports.AdvisorSpec,
) (spec *llmports.AdvisorSpec, fromSetting bool) {
	if requested != nil {
		clone := *requested

		return &clone, false
	}
	if model := provider.ResolvedAdvisorModel(); model != "" {
		return llmports.DefaultAdvisor(model), true
	}

	return nil, false
}

// refusedAdvisor reports a call the API refused because of the advisor it
// declared, such as an advisor less capable than the model it advises.
func refusedAdvisor(req *modeladapter.Request, err error) (string, bool) {
	if err == nil || req.Advisor == nil {
		return "", false
	}

	var transport *modeladapter.TransportError
	if !errors.As(err, &transport) || transport.StatusCode != http.StatusBadRequest {
		return "", false
	}
	if !strings.Contains(strings.ToLower(transport.Message), advisorField) {
		return "", false
	}

	return transport.Message, true
}
