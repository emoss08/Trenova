package formula

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/environment"
	formulaerrors "github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"go.uber.org/zap"
)

const CodeUnresolvedField = "unresolvedField"

type RateRequest struct {
	Template        *formulatemplate.FormulaTemplate
	Entity          any
	Variables       map[string]any
	Overrides       map[string]any
	RatingDate      int64
	Unavailable     map[string]string
	TraceStatements bool
}

type Rated struct {
	Response  *formulatemplatetypes.CalculateResponse
	Evaluated *Evaluated
	Env       map[string]any
}

func (s *Service) Rate(
	ctx context.Context,
	req *RateRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	rated, err := s.RateDetailed(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = applyPolicy(rated, req.Template.ChargePolicy()); err != nil {
		return nil, err
	}
	return rated.Response, nil
}

func (s *Service) RateDetailed(ctx context.Context, req *RateRequest) (*Rated, error) {
	started := time.Now()

	tenantInfo := pagination.TenantInfo{
		OrgID: req.Template.OrganizationID,
		BuID:  req.Template.BusinessUnitID,
	}

	if err := s.validateVariableKeys(ctx, req.Template, req.Variables); err != nil {
		return nil, err
	}

	built, err := s.env.Build(
		req.Entity,
		req.Template.SchemaID,
		s.providedValues(ctx, tenantInfo, req.Entity, req.RatingDate),
		req.Variables,
	)
	if err != nil {
		return nil, formulaerrors.NewSchemaError(req.Template.SchemaID, "build environment", err)
	}
	if err = environment.Merge(
		built.Env,
		req.Overrides,
		built.Sources,
		formulatypes.ValueSourceOverride,
	); err != nil {
		return nil, errortypes.NewValidationError("overrides", errortypes.ErrInvalid, err.Error())
	}
	if err = applyDefaults(req, built); err != nil {
		return nil, err
	}
	unresolved, err := s.unresolvedReads(ctx, req, built)
	if err != nil {
		return nil, err
	}
	if len(unresolved) > 0 && len(unresolved[0]) > 0 {
		return nil, s.unresolvedError(unresolved[0], built.Failures)
	}

	evaluated, err := s.evaluate(ctx, &evaluation{
		tenant:          tenantInfo,
		env:             built.Env,
		expression:      req.Template.Expression,
		breakdowns:      req.Template.BreakdownDefinitions,
		traceStatements: req.TraceStatements,
	})
	if err != nil {
		return nil, err
	}
	if evaluated.Error != nil {
		return nil, s.ratingError(evaluated.Error, built.Failures, req.Unavailable)
	}
	for index := range evaluated.Lines {
		if index+1 < len(unresolved) && len(unresolved[index+1]) > 0 {
			evaluated.Lines[index].Error = UnresolvedFormulaError(unresolved[index+1])
		}
	}

	receipt := evaluated.Receipt(environment.ReceiptVariables(built.Env, built.Sources))
	receipt.VersionNumber = req.Template.CurrentVersionNumber
	receipt.DurationMicros = time.Since(started).Microseconds()

	return &Rated{
		Evaluated: evaluated,
		Env:       built.Env,
		Response: &formulatemplatetypes.CalculateResponse{
			Amount:              evaluated.Amount,
			Variables:           built.Env,
			FormulaTemplateID:   req.Template.ID.String(),
			FormulaTemplateName: req.Template.Name,
			Expression:          req.Template.Expression,
			VersionNumber:       req.Template.CurrentVersionNumber,
			Receipt:             receipt,
		},
	}, nil
}

func (s *Service) unresolvedReads(
	ctx context.Context,
	req *RateRequest,
	built *environment.Built,
) ([][]string, error) {
	for path := range built.Failures {
		if environment.ValueAt(built.Env, path) != nil {
			delete(built.Failures, path)
		}
	}
	if len(built.Failures) == 0 {
		return nil, nil
	}

	variables, err := s.SchemaVariables(req.Template.SchemaID)
	if err != nil {
		return nil, err
	}
	schema := &rules.FormulaSchema{
		Variables: variables,
		Inputs:    TemplateInputs(req.Template.VariableDefinitions),
	}

	sources := make([]string, 0, len(req.Template.BreakdownDefinitions)+1)
	sources = append(sources, req.Template.Expression)
	for _, def := range req.Template.BreakdownDefinitions {
		if def != nil {
			sources = append(sources, def.Expression)
		}
	}

	reads := make([][]string, 0, len(sources))
	for _, source := range sources {
		check, checkErr := rules.CheckFormula(ctx, &rules.CheckRequest{
			Source: source,
			Expect: rules.ExpectAmount,
			Schema: schema,
		})
		if checkErr != nil {
			return nil, checkErr
		}
		reads = append(reads, fieldsIn(check.SchemaReads, built.Failures))
	}
	return reads, nil
}

func UnresolvedFormulaError(fields []string) *rules.FormulaError {
	return &rules.FormulaError{
		Code:    CodeUnresolvedField,
		Message: formulaerrors.NewUnresolvedFieldError(fields, nil).Text(),
		Fields:  fields,
	}
}

func applyPolicy(rated *Rated, policy formulatypes.ChargePolicy) error {
	resp := rated.Response
	amount, guardrail, rounding, err := ApplyChargePolicy(policy, rated.Evaluated.Amount)
	resp.Amount = amount
	resp.Guardrail = guardrail
	resp.Rounding = rounding
	resp.Breakdown = RoundBreakdown(
		policy,
		rated.Evaluated.Breakdown(i18n.Default),
		rated.Evaluated.Amount,
	)
	return err
}

func applyDefaults(req *RateRequest, built *environment.Built) error {
	for name := range req.Unavailable {
		if _, supplied := built.Sources[name]; !supplied {
			built.Env[name] = nil
		}
	}

	filled := make([]string, 0, len(req.Template.VariableDefinitions))
	for _, def := range req.Template.VariableDefinitions {
		if def == nil {
			continue
		}
		if _, unavailable := req.Unavailable[def.Name]; unavailable {
			continue
		}
		if _, exists := built.Env[def.Name]; exists || def.DefaultValue == nil {
			continue
		}
		value, err := environment.Exact(def.DefaultValue)
		if err != nil {
			return errortypes.NewValidationError(
				"variableDefinitions",
				errortypes.ErrInvalid,
				"The default of {0} is not a usable number",
				def.Name,
			)
		}
		built.Env[def.Name] = value
		filled = append(filled, def.Name)
	}
	built.Sources.Mark(filled, formulatypes.ValueSourceDefault)
	return nil
}

func (s *Service) ratingError(
	evalErr *rules.FormulaError,
	failures map[string]error,
	unavailable map[string]string,
) error {
	if evalErr.Code == rules.ErrorMissingField {
		for _, field := range evalErr.Fields {
			if message, ok := unavailable[field]; ok {
				return errortypes.NewValidationError(
					"formulaTemplateId",
					errortypes.ErrInvalid,
					message,
				)
			}
		}
		if unresolvedFields := fieldsIn(evalErr.Fields, failures); len(unresolvedFields) > 0 {
			return s.unresolvedError(unresolvedFields, failures)
		}
	}

	return EvaluationValidationError(MainScope, evalErr)
}

func (s *Service) unresolvedError(fields []string, failures map[string]error) error {
	causes := make(map[string]error, len(fields))
	for _, field := range fields {
		causes[field] = failures[field]
		s.l.Warn("formula reads a field the record could not supply",
			append(
				formulaerrors.LogFields(failures[field]),
				zap.String("field", field),
				zap.Error(failures[field]),
			)...,
		)
	}
	unresolved := formulaerrors.NewUnresolvedFieldError(fields, causes)
	message, args := unresolved.LocalizedMessage()
	return &errortypes.Error{
		Field:    MainScope,
		Code:     errortypes.ErrInvalid,
		Message:  message,
		Args:     args,
		Internal: unresolved,
	}
}

func EvaluationValidationError(field string, evalErr *rules.FormulaError) *errortypes.Error {
	return &errortypes.Error{
		Field:    field,
		Code:     errortypes.ErrInvalid,
		Message:  evalErr.Message.Template,
		Args:     evalErr.Message.Args,
		Internal: formulaerrors.NewEvaluationError(field, evalErr),
	}
}

func (s *Service) validateVariableKeys(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	variables map[string]any,
) error {
	if len(variables) == 0 {
		return nil
	}

	if err := refuseReserved(ctx, variables); err != nil {
		return err
	}

	declared := make(map[string]struct{}, len(template.VariableDefinitions))
	for _, varDef := range template.VariableDefinitions {
		if varDef != nil {
			declared[varDef.Name] = struct{}{}
		}
	}
	roots := s.env.FieldRoots(template.SchemaID)

	for key := range variables {
		if _, ok := declared[key]; ok {
			continue
		}
		root, _, _ := strings.Cut(key, ".")
		if _, shadows := roots[root]; shadows {
			return errortypes.NewValidationError(
				"variables",
				errortypes.ErrInvalid,
				"{0} is already a shipment value; an input cannot replace it, so choose another name",
				key,
			)
		}
	}

	return nil
}

func refuseReserved(ctx context.Context, variables map[string]any) error {
	if len(variables) == 0 {
		return nil
	}

	catalog, err := rules.Catalog(ctx)
	if err != nil {
		return err
	}

	for key := range variables {
		if catalog.IsReserved(key) {
			return errortypes.NewValidationError(
				"variables",
				errortypes.ErrInvalid,
				"{0} is the name of a formula function; choose another name for the input",
				key,
			)
		}
	}
	return nil
}
