package completionengine

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	maxAdvisorRefusalChars = 1000
)

// recordAdvisorRefusal keeps the API's refusal on the provider, where AI
// control shows it, so the setting is not failing out of sight. Until the
// advisor or the model changes, calls to the provider go out without it.
func (s *Engine) RecordAdvisorRefusal(
	ctx context.Context,
	provider *aiprovider.Provider,
	tenant pagination.TenantInfo,
	advisor string,
	message string,
) {
	s.Logger.Warn("the provider refused the advisor its settings declare; answering without it",
		zap.String("provider", provider.ID.String()),
		zap.String("model", provider.Model),
		zap.String("advisor", advisor),
		zap.String("message", message),
	)
	if s.Advisors == nil || tenant.OrgID.IsNil() {
		return
	}
	if runes := []rune(message); len(runes) > maxAdvisorRefusalChars {
		message = string(runes[:maxAdvisorRefusalChars])
	}

	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), slotStoreTimeout)
	defer cancel()
	if err := s.Advisors.RecordAdvisorRefusal(recordCtx, &aiproviderports.RecordAdvisorRefusalRequest{
		ID:         provider.ID,
		TenantInfo: tenant,
		Refusal: aiprovider.AdvisorRefusal{
			Advisor:   advisor,
			Model:     provider.Model,
			Message:   message,
			RefusedAt: time.Now().Unix(),
		},
	}); err != nil {
		s.Logger.Warn("could not record the provider's advisor refusal",
			zap.String("provider", provider.ID.String()),
			zap.Error(err),
		)
	}
}
