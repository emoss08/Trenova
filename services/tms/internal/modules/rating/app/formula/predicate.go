package formula

import (
	"context"
	"strconv"

	formulaerrors "github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/rules"
	"go.uber.org/zap"
)

func (s *Service) EvaluatePredicates(
	ctx context.Context,
	req *formulatemplateports.EvaluatePredicatesRequest,
) ([]formulatemplateports.PredicateOutcome, error) {
	if len(req.Expressions) == 0 {
		return nil, nil
	}

	built, err := s.env.Build(
		req.Entity,
		req.SchemaID,
		s.providedValues(ctx, req.TenantInfo, req.Entity, 0),
		nil,
	)
	if err != nil {
		return nil, formulaerrors.NewSchemaError(req.SchemaID, "build environment", err)
	}

	programs := make([]rules.FormulaProgram, 0, len(req.Expressions))
	for index, expression := range req.Expressions {
		programs = append(programs, rules.FormulaProgram{
			ID:     strconv.Itoa(index),
			Source: expression,
			Expect: rules.ExpectPredicate,
		})
	}

	resp, err := rules.EvaluateFormulas(ctx, &rules.EvaluateRequest{
		Programs: programs,
		Env:      built.Env,
	})
	if err != nil {
		s.l.Warn("failed to evaluate predicates",
			zap.String("schemaId", req.SchemaID),
			zap.Int("count", len(req.Expressions)),
			zap.Error(err),
		)
		return nil, err
	}

	outcomes := make([]formulatemplateports.PredicateOutcome, len(req.Expressions))
	for index := range resp.Results {
		result := &resp.Results[index]
		if result.Error != nil {
			outcomes[index].Err = s.ratingError(result.Error, built.Failures, nil)
			continue
		}
		outcomes[index].Holds, outcomes[index].Err = result.Holds()
	}

	return outcomes, nil
}
