package completionrouter

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	advisorField           = "advisor"
	maxAdvisorRefusalChars = 1000
)

// advisorFor is the advisor a chat call to provider declares. A caller that
// names one, such as a desk bench cell, overrides the provider's setting;
// otherwise the provider's own setting decides.
func advisorFor(
	provider *aiprovider.Provider,
	requested *agentports.AdvisorSpec,
) (spec *agentports.AdvisorSpec, fromSetting bool) {
	if requested != nil {
		clone := *requested

		return &clone, false
	}
	if model := provider.ResolvedAdvisorModel(); model != "" {
		return agentports.DefaultAdvisor(model), true
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

// recordAdvisorRefusal keeps the API's refusal on the provider, where AI
// control shows it, so the setting is not failing out of sight. Until the
// advisor or the model changes, calls to the provider go out without it.
func (s *Service) recordAdvisorRefusal(
	ctx context.Context,
	provider *aiprovider.Provider,
	tenant pagination.TenantInfo,
	advisor string,
	message string,
) {
	s.logger.Warn("the provider refused the advisor its settings declare; answering without it",
		zap.String("provider", provider.ID.String()),
		zap.String("model", provider.Model),
		zap.String("advisor", advisor),
		zap.String("message", message),
	)
	if s.advisors == nil || tenant.OrgID.IsNil() {
		return
	}
	if runes := []rune(message); len(runes) > maxAdvisorRefusalChars {
		message = string(runes[:maxAdvisorRefusalChars])
	}

	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), slotStoreTimeout)
	defer cancel()
	if err := s.advisors.RecordAdvisorRefusal(recordCtx, &agentports.RecordAdvisorRefusalRequest{
		ID:         provider.ID,
		TenantInfo: tenant,
		Refusal: aiprovider.AdvisorRefusal{
			Advisor:   advisor,
			Model:     provider.Model,
			Message:   message,
			RefusedAt: time.Now().Unix(),
		},
	}); err != nil {
		s.logger.Warn("could not record the provider's advisor refusal",
			zap.String("provider", provider.ID.String()),
			zap.Error(err),
		)
	}
}
