package shipmentnarration

import (
	"context"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Narration[T any] struct {
	Draft T
	Model string
}

func Narrate[T any](
	ctx context.Context,
	completion services.CompletionService,
	logger *zap.Logger,
	req *services.StructuredCompletionRequest,
) (*Narration[T], bool) {
	if completion == nil {
		return nil, false
	}

	result, err := completion.CompleteStructured(ctx, req)
	if err != nil {
		if errors.Is(err, services.ErrNoProviderConfigured) {
			logger.Debug("no provider configured; using computed wording",
				zap.String("schema", req.SchemaName))
			return nil, false
		}
		logger.Warn("narration failed; using computed wording",
			zap.String("schema", req.SchemaName),
			zap.Error(err),
		)
		return nil, false
	}

	out := &Narration[T]{Model: result.ModelIdentifier}
	if err = sonic.UnmarshalString(result.Text, &out.Draft); err != nil {
		logger.Warn("narration could not be parsed; using computed wording",
			zap.String("schema", req.SchemaName),
			zap.String("model", result.ModelIdentifier),
			zap.Error(err),
		)
		return nil, false
	}

	return out, true
}

func Supported(prose string, supported []decimal.Decimal) bool {
	return numberguard.CheckNumbers(prose, supported).OK
}

func IntValues(values ...int) []decimal.Decimal {
	out := make([]decimal.Decimal, 0, len(values))
	for _, value := range values {
		out = append(out, decimal.NewFromInt(int64(value)))
	}
	return out
}
