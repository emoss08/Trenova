package formula

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/environment"
	formulaerrors "github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
)

type PreviewRequest struct {
	SchemaID        string
	Expression      string
	Breakdowns      []*formulatypes.BreakdownDefinition
	Variables       map[string]any
	Entity          any
	TenantInfo      pagination.TenantInfo
	TraceStatements bool
}

type ScopedDiagnostic struct {
	Scope string
	rules.FormulaDiagnostic
}

//nolint:gocritic // hugeParam: a value receiver, so a value held by value marshals through it
func (d ScopedDiagnostic) MarshalJSON() ([]byte, error) {
	return d.AppendJSON(make([]byte, 0, 256), d.Scope)
}

type Preview struct {
	Diagnostics []ScopedDiagnostic
	Statements  []formulatemplatetypes.Statement
	Evaluated   *Evaluated
	Error       *formulaerrors.EvaluationError
	Variables   map[string]any
	Receipt     *formulatypes.Receipt
}

func (p *Preview) Accepted() bool {
	for index := range p.Diagnostics {
		if p.Diagnostics[index].IsError() {
			return false
		}
	}
	return true
}

func (p *Preview) Valid() bool {
	return p.Accepted() && p.Evaluated != nil && p.Error == nil
}

func PreviewScope(index int) string {
	return fmt.Sprintf("breakdowns[%d].expression", index)
}

func (s *Service) Preview(ctx context.Context, req *PreviewRequest) (*Preview, error) {
	if err := refuseReserved(ctx, req.Variables); err != nil {
		return nil, err
	}

	sources := make([]CheckSource, 0, len(req.Breakdowns)+1)
	sources = append(sources, CheckSource{Scope: MainScope, Source: req.Expression})
	for index, def := range req.Breakdowns {
		if def != nil {
			sources = append(
				sources,
				CheckSource{Scope: PreviewScope(index), Source: def.Expression},
			)
		}
	}

	checked, err := s.Check(ctx, &CheckRequest{
		SchemaID:   req.SchemaID,
		Inputs:     ValueInputs(sampleInputs(req.Variables, s.env.FieldRoots(req.SchemaID))),
		TenantInfo: req.TenantInfo,
		Expect:     rules.ExpectAmount,
		Sources:    sources,
	})
	if err != nil {
		return nil, err
	}

	preview := &Preview{}
	for _, item := range checked {
		for index := range item.Check.Diagnostics {
			preview.Diagnostics = append(preview.Diagnostics, ScopedDiagnostic{
				Scope:             item.Scope,
				FormulaDiagnostic: item.Check.Diagnostics[index],
			})
		}
	}
	if !preview.Accepted() {
		preview.Statements = splitStatements(checked[0].Check)
		return preview, nil
	}

	built, err := s.previewEnvironment(ctx, req)
	if err != nil {
		return nil, err
	}

	evaluated, err := s.evaluate(ctx, &evaluation{
		tenant:          req.TenantInfo,
		env:             built.Env,
		expression:      req.Expression,
		breakdowns:      req.Breakdowns,
		traceStatements: req.TraceStatements,
	})
	if err != nil {
		return nil, err
	}

	markUnresolved(evaluated, checked, built)

	preview.Evaluated = evaluated
	preview.Statements = evaluated.Statements
	preview.Variables = built.Env
	preview.Receipt = evaluated.Receipt(environment.ReceiptVariables(built.Env, built.Sources))
	preview.Error = firstEvaluationError(evaluated)
	if !req.TraceStatements {
		preview.Statements = nil
	}

	return preview, nil
}

func (s *Service) previewEnvironment(
	ctx context.Context,
	req *PreviewRequest,
) (*environment.Built, error) {
	provided := s.providedValues(ctx, req.TenantInfo, req.Entity, 0)

	var (
		built *environment.Built
		err   error
	)
	if req.Entity != nil {
		built, err = s.env.Build(req.Entity, req.SchemaID, provided, req.Variables)
	} else {
		built, err = s.env.Zero(req.SchemaID, provided, req.Variables)
	}
	if err != nil {
		return nil, errortypes.NewValidationError("variables", errortypes.ErrInvalid, err.Error())
	}
	return built, nil
}

func markUnresolved(evaluated *Evaluated, checked []Checked, built *environment.Built) {
	failures := built.Failures
	for path := range failures {
		if environment.ValueAt(built.Env, path) != nil {
			delete(failures, path)
		}
	}
	if len(failures) == 0 {
		return
	}
	if fields := fieldsIn(checked[0].Check.SchemaReads, failures); len(fields) > 0 {
		evaluated.Error = UnresolvedFormulaError(fields)
	}
	for index := range evaluated.Lines {
		if index+1 >= len(checked) {
			break
		}
		if fields := fieldsIn(checked[index+1].Check.SchemaReads, failures); len(fields) > 0 {
			evaluated.Lines[index].Error = UnresolvedFormulaError(fields)
		}
	}
}

func sampleInputs(variables map[string]any, roots map[string]struct{}) map[string]any {
	inputs := make(map[string]any, len(variables))
	for name, value := range variables {
		if _, field := roots[name]; field || strings.Contains(name, ".") {
			continue
		}
		inputs[name] = value
	}
	return inputs
}

func firstEvaluationError(evaluated *Evaluated) *formulaerrors.EvaluationError {
	if evaluated.Error != nil {
		return formulaerrors.NewEvaluationError(MainScope, evaluated.Error)
	}
	for _, line := range evaluated.Lines {
		if line.Error != nil {
			return formulaerrors.NewEvaluationError(PreviewScope(line.Index), line.Error)
		}
	}
	return nil
}

func splitStatements(check *rules.FormulaCheck) []formulatemplatetypes.Statement {
	statements := make([]formulatemplatetypes.Statement, 0, len(check.Statements))
	for _, statement := range check.Statements {
		statements = append(statements, formulatemplatetypes.Statement{
			Index: statement.Index,
			Kind:  statement.Kind,
			Name:  statement.Name,
			Code:  statement.Code,
			From:  statement.From,
			To:    statement.To,
		})
	}
	return statements
}
