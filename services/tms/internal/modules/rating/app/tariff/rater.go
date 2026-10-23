package tariff

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ProgramTariff   = "tariff"
	conditionPrefix = "condition:"

	errorCodeInput = "input"
)

var (
	_ tariffports.Rater = (*Rater)(nil)

	errMissingResult = errors.New("tariff: the module answered without a program's result")
)

type evaluator func(ctx context.Context, req *rules.TariffEvaluateRequest) (*rules.TariffEvaluation, error)

type RaterParams struct {
	fx.In

	Logger   *zap.Logger
	Contexts tariffports.ContextSource
}

type Rater struct {
	l        *zap.Logger
	contexts tariffports.ContextSource
	evaluate evaluator
}

func NewRater(p RaterParams) *Rater {
	return &Rater{
		l:        p.Logger.Named("tariff.rater"),
		contexts: p.Contexts,
		evaluate: rules.EvaluateTariffs,
	}
}

func NewRaterWith(
	logger *zap.Logger,
	contexts tariffports.ContextSource,
	evaluate func(context.Context, *rules.TariffEvaluateRequest) (*rules.TariffEvaluation, error),
) *Rater {
	return &Rater{l: logger, contexts: contexts, evaluate: evaluate}
}

func ConditionProgramID(id string) string {
	return conditionPrefix + id
}

func (r *Rater) Rate(
	ctx context.Context,
	req *tariffports.RateRequest,
) (*tariffports.RateResult, error) {
	var libraries []rules.TariffLibrary
	if req.Template != nil {
		libraries = req.Template.Libraries
	}
	tariffCtx, err := r.contexts.Context(ctx, &tariffports.ContextRequest{
		TenantInfo: req.TenantInfo,
		Libraries:  libraries,
	})
	if err != nil {
		return nil, err
	}

	evalReq := &rules.TariffEvaluateRequest{
		Context:  tariffCtx,
		Record:   req.Record,
		Programs: programs(req),
		Trace:    req.Trace,
	}
	if len(evalReq.Programs) == 0 {
		return &tariffports.RateResult{Conditions: map[string]bool{}}, nil
	}

	evaluation, err := r.evaluateWithCells(ctx, req, evalReq)
	if err != nil {
		return nil, err
	}

	return r.result(req, evaluation)
}

func programs(req *tariffports.RateRequest) []rules.TariffProgram {
	out := make([]rules.TariffProgram, 0, len(req.Conditions)+1)
	if req.Template != nil {
		out = append(out, rules.TariffProgram{
			ID:       ProgramTariff,
			Source:   req.Template.Source,
			Kind:     rules.TariffKindTariff,
			Inputs:   declaredInputs(req.Template.Inputs, req.Inputs),
			Exchange: req.Exchange,
		})
	}
	for i := range req.Conditions {
		condition := &req.Conditions[i]
		out = append(out, rules.TariffProgram{
			ID:     ConditionProgramID(condition.ID),
			Source: condition.Source,
			Kind:   rules.TariffKindCondition,
			Inputs: condition.Inputs,
		})
	}
	return out
}

func declaredInputs(
	decls []tarifftypes.InputDecl,
	inputs tarifftypes.Inputs,
) map[string]rules.RawJSON {
	if len(inputs) == 0 {
		return nil
	}
	if len(decls) == 0 {
		return inputs
	}
	out := make(map[string]rules.RawJSON, len(decls))
	for i := range decls {
		if value, ok := inputs[decls[i].Name]; ok {
			out[decls[i].Name] = value
		}
	}
	return out
}

func (r *Rater) evaluateWithCells(
	ctx context.Context,
	req *tariffports.RateRequest,
	evalReq *rules.TariffEvaluateRequest,
) (*rules.TariffEvaluation, error) {
	loads := &cellLoad{contexts: r.contexts, tenant: req.TenantInfo, tariffCtx: evalReq.Context}
	var evaluation *rules.TariffEvaluation
	err := loads.until(ctx, func() ([]rules.TariffNeed, error) {
		var evalErr error
		evaluation, evalErr = r.evaluate(ctx, evalReq)
		if evalErr != nil {
			return nil, evalErr
		}
		return collectNeeds(evaluation), nil
	})
	if err != nil {
		return nil, err
	}
	return evaluation, nil
}

func collectNeeds(evaluation *rules.TariffEvaluation) []rules.TariffNeed {
	var needs []rules.TariffNeed
	for i := range evaluation.Results {
		if evaluation.Results[i].Status == rules.TariffNeeds {
			needs = append(needs, evaluation.Results[i].Needs...)
		}
	}
	return needs
}

func (r *Rater) result(
	req *tariffports.RateRequest,
	evaluation *rules.TariffEvaluation,
) (*tariffports.RateResult, error) {
	out := &tariffports.RateResult{Conditions: make(map[string]bool, len(req.Conditions))}

	for i := range req.Conditions {
		id := req.Conditions[i].ID
		result := evaluation.Result(ConditionProgramID(id))
		if result == nil {
			return nil, errMissingResult
		}
		holds := result.Status == rules.TariffRated && result.Value != nil && *result.Value
		if result.Status != rules.TariffRated {
			r.l.Warn("an accessorial condition did not evaluate; the charge is withheld",
				zap.String("condition", id),
				zap.String("status", string(result.Status)),
			)
		}
		out.Conditions[id] = holds
	}

	if req.Template == nil {
		return out, nil
	}
	result := evaluation.Result(ProgramTariff)
	if result == nil {
		return nil, errMissingResult
	}
	out.Trace = result.Trace
	out.Lookups = result.Lookups
	out.Choices = result.Choices

	switch result.Status {
	case rules.TariffRated:
		bill, err := tarifftypes.BillFrom(result.Bill, req.Currency)
		if err != nil {
			return nil, err
		}
		out.Bill = bill
		return out, nil
	case rules.TariffRefused:
		out.Refusal = result.Refusal
		return out, nil
	case rules.TariffError:
		return nil, evaluationError(result.Error)
	case rules.TariffNeeds:
		return nil, fmt.Errorf("%w: status %q", errMissingResult, result.Status)
	default:
		return nil, fmt.Errorf("%w: status %q", errMissingResult, result.Status)
	}
}

func evaluationError(evalErr *rules.TariffEvalError) error {
	if evalErr == nil {
		return errMissingResult
	}
	if evalErr.Code == errorCodeInput {
		return evalErr.Message.ValidationError("inputs", errortypes.ErrInvalid)
	}
	return evalErr.Message.BusinessError().
		WithParam("code", evalErr.Code).
		WithInternal(fmt.Errorf("tariff evaluation failed with code %s", evalErr.Code))
}
