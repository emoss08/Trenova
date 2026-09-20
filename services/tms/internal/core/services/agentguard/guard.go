package agentguard

import (
	"context"
	"errors"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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

	// verdicts remembers classifications already made, so the same question
	// asked twice costs one call rather than two.
	verdicts *verdictCache

	// RefuseWhenUnavailable restores the old posture: a classifier that cannot
	// be reached refuses the request rather than falling back to the
	// deterministic verdict. Off by default, because the deterministic layer is
	// what this would be falling back to and an install with no classifier
	// configured runs on exactly that, permanently and by design.
	RefuseWhenUnavailable bool
}

func New(p Params) *Service {
	return &Service{
		logger:     p.Logger.Named("service.agent-guard"),
		completion: p.Completion,
		verdicts:   newVerdictCache(),
	}
}

// Evaluate decides whether a request may proceed.
//
// The deterministic layer runs first because it is free and catches the blatant
// cases. Anything it does not recognise goes to the classifier, whose verdict
// decides.
//
// The classifier is shown the tail of the conversation, because scope is not a
// property of a sentence on its own. "Can you give me a link to download it?"
// carries no subject at all: read alone it is unclassifiable, and the classifier
// duly refused it one turn after running the report it was asking about. The
// referent lives in the previous turns, so those go with it. Only the latest
// message is judged — earlier turns are background, never instruction, and
// never the thing being classified.
//
// A classifier that cannot produce a verdict — for any reason — leaves the
// request on the deterministic verdict alone, recorded as StageUnavailable so
// the degradation is visible in the thread rather than inferred.
//
// This used to distinguish "no provider configured" (proceed) from "the
// configured provider failed" (refuse), on the reasoning that a control which
// silently stops working is how guardrails rot. The distinction does not
// survive contact with what the two states actually are. In both, the control
// is not operating; the only difference is whether a row exists. Treating one
// as fine and the other as fatal made the product less reliable the more of it
// you configured, and it showed: on a single flaky provider, "which drivers
// have a medical card expiring in the next 360 days" was refused outright
// twenty-six seconds after the same question had been answered.
//
// Nothing about tenant safety rests here. The deterministic rules have already
// run and passed, the system prompt still refuses off-domain and software work,
// and every tool call is authorized against the acting user independently. This
// layer filters scope, and failing it closed denies service rather than
// protecting anything. An operator who wants the stricter posture can set
// RefuseWhenUnavailable.
func (s *Service) Evaluate(ctx context.Context, req EvaluateRequest) Decision {
	if decision := EvaluateDeterministic(req.Input); !decision.Allowed {
		s.logger.Info("request refused by deterministic scope rule",
			zap.String("rule", decision.MatchedRule),
			zap.String("reason", string(decision.Reason)),
		)

		return decision
	}

	result, err := s.Classify(ctx, req)
	if err != nil {
		if errors.Is(err, serviceports.ErrNoProviderConfigured) {
			s.logger.Debug("no scope classifier configured; deterministic rules only")

			return allowed(StageUnavailable, CategoryOther)
		}

		if s.RefuseWhenUnavailable {
			s.logger.Warn("scope classifier failed; refusing request", zap.Error(err))

			return refused(
				StageUnavailable,
				ReasonClassifierUnavailable,
				CategoryOther,
				"classifier_error",
			)
		}

		// Warn, not Debug: an operator who configured a classifier should be
		// able to see that it is not running, even though the request proceeds.
		s.logger.Warn("scope classifier unavailable; falling back to deterministic rules",
			zap.Error(err),
		)

		return allowed(StageUnavailable, CategoryOther)
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
