package aiproviderservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// advisorCheck is a save that needs its advisor confirmed against the
// endpoint's own model list: the advisor was turned on, named anew, or now
// advises another model or another endpoint.
type advisorCheck struct {
	before *aiprovider.Provider
	after  *aiprovider.Provider
	req    *aiproviderports.SaveAIProviderRequest
}

func (c *advisorCheck) needed() bool {
	if !c.after.AdvisorEnabled || c.after.CheckAdvisor() != nil {
		return false
	}

	return c.before == nil || aiprovider.AdvisorSettingsChanged(c.before, c.after) ||
		!aiprovider.SameCredentialTarget(c.before, c.after)
}

// confirmAdvisor refuses an advisor the provider's endpoint does not list.
// Whether that advisor may advise the provider's model is the API's to say;
// it says so on the first call, and the router records it on the provider.
func (s *Service) confirmAdvisor(
	ctx context.Context,
	check *advisorCheck,
	multiErr *errortypes.MultiError,
) {
	if !check.needed() || s.tester == nil {
		return
	}

	advisor := check.after.AdvisorModelName()
	options, err := s.ListModels(ctx, &aiproviderports.ListAIProviderModelsRequest{
		TenantInfo: check.req.TenantInfo,
		Endpoint: aiproviderports.AIProviderEndpoint{
			ProviderID:          check.req.ID,
			Kind:                check.after.Kind,
			Platform:            check.after.Platform,
			PlatformSettings:    check.after.PlatformSettings,
			BaseURL:             check.after.BaseURL,
			APIKey:              check.req.APIKey,
			AllowPrivateNetwork: check.after.AllowPrivateNetwork,
		},
	})
	if err != nil {
		multiErr.Add("advisorModel", errortypes.ErrInvalid,
			"The provider's model list could not be read to confirm the advisor: "+err.Error())
		return
	}
	for idx := range options {
		if strings.EqualFold(options[idx].ID, advisor) {
			return
		}
	}
	multiErr.Add("advisorModel", errortypes.ErrInvalid,
		"The provider's model list does not include "+advisor)
}
