package formula_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

type evaluationRequest struct {
	Template  *formulatemplate.FormulaTemplate
	Entity    any
	Variables map[string]any
}

type evaluationResult struct {
	Value     decimal.Decimal
	Variables map[string]any
	Breakdown []formulatemplatetypes.BreakdownAmount
	Receipt   *formulatypes.Receipt
}

type rawRating struct {
	svc *formula.Service
}

func newRawRating(t *testing.T) *rawRating {
	t.Helper()
	return &rawRating{svc: setupService(t)}
}

func (r *rawRating) Evaluate(
	ctx context.Context,
	req *evaluationRequest,
) (*evaluationResult, error) {
	rated, err := r.svc.RateDetailed(ctx, &formula.RateRequest{
		Template:  req.Template,
		Entity:    req.Entity,
		Variables: req.Variables,
	})
	if err != nil {
		return nil, err
	}
	return &evaluationResult{
		Value:     rated.Evaluated.Amount,
		Variables: rated.Env,
		Breakdown: rated.Evaluated.Breakdown(i18n.Default),
		Receipt:   rated.Response.Receipt,
	}, nil
}
