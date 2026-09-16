package agentguard

import (
	"context"
	"errors"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger     *zap.Logger
	Completion serviceports.CompletionService
}

type Service struct {
	logger     *zap.Logger
	completion serviceports.CompletionService
}

func New(p Params) *Service {
	return &Service{
		logger:     p.Logger.Named("service.agent-guard"),
		completion: p.Completion,
	}
}

// Evaluate decides whether a request may proceed.
//
// The deterministic layer runs first because it is free and catches the blatant
// cases. Anything it does not recognise goes to the classifier, whose verdict
// decides. If no classifier provider is configured the request proceeds on the
// deterministic verdict alone — the assistant still refuses in its own system
// prompt, and breaking chat entirely because an optional provider is unset would
// be a worse outcome than a weaker filter. If a classifier *is* configured and
// the call fails, the request is refused, because a configured control that
// silently stops working is how guardrails rot.
func (s *Service) Evaluate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	input string,
) Decision {
	if decision := EvaluateDeterministic(input); !decision.Allowed {
		s.logger.Info("request refused by deterministic scope rule",
			zap.String("rule", decision.MatchedRule),
			zap.String("reason", string(decision.Reason)),
		)

		return decision
	}

	result, err := s.Classify(ctx, tenantInfo, input)
	if err != nil {
		if errors.Is(err, serviceports.ErrNoProviderConfigured) {
			s.logger.Debug("no scope classifier configured; deterministic rules only")

			return allowed(StageUnavailable, CategoryOther)
		}

		s.logger.Warn("scope classifier failed; refusing request", zap.Error(err))

		return refused(
			StageUnavailable,
			ReasonClassifierUnavailable,
			CategoryOther,
			"classifier_error",
		)
	}

	category := Category(result.Category)
	if category.InScope() {
		return allowed(StageClassifier, category)
	}

	s.logger.Info("request refused by scope classifier",
		zap.String("category", result.Category),
		zap.String("reasoning", result.Reasoning),
	)

	return refused(StageClassifier, reasonForCategory(category), category, "")
}

func reasonForCategory(category Category) Reason {
	switch category {
	case CategoryCodeGeneration:
		return ReasonCodeGeneration
	case CategoryPromptManipulation:
		return ReasonPromptManipulation
	default:
		return ReasonOffDomain
	}
}
