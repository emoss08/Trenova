package formula

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
)

type CheckRequest struct {
	SchemaID   string
	Inputs     []rules.FormulaInput
	TenantInfo pagination.TenantInfo
	Expect     rules.FormulaExpect
	Sources    []CheckSource
}

type CheckSource struct {
	Scope  string
	Source string
}

type Checked struct {
	Scope string
	Check *rules.FormulaCheck
}

func (s *Service) Check(ctx context.Context, req *CheckRequest) ([]Checked, error) {
	variables, err := s.SchemaVariables(req.SchemaID)
	if err != nil {
		return nil, err
	}

	texts := make([]string, 0, len(req.Sources))
	for _, source := range req.Sources {
		texts = append(texts, source.Source)
	}
	selection, err := s.tablesFor(ctx, req.TenantInfo, texts...)
	if err != nil {
		return nil, err
	}

	schema := &rules.FormulaSchema{Variables: variables, Inputs: req.Inputs}
	checked := make([]Checked, 0, len(req.Sources))
	for _, source := range req.Sources {
		result, checkErr := rules.CheckFormula(ctx, &rules.CheckRequest{
			Source: source.Source,
			Expect: req.Expect,
			Schema: schema,
			Tables: shapesOf(selection),
		})
		if checkErr != nil {
			return nil, checkErr
		}
		checked = append(checked, Checked{Scope: source.Scope, Check: result})
	}

	return checked, nil
}

func shapesOf(selection *formulatemplatetypes.TableSelection) []rules.FormulaTable {
	if selection == nil {
		return nil
	}
	return selection.Shapes
}

func TemplateInputs(definitions []*formulatypes.VariableDefinition) []rules.FormulaInput {
	inputs := make([]rules.FormulaInput, 0, len(definitions))
	for _, def := range definitions {
		if def == nil || def.Name == "" {
			continue
		}
		inputs = append(inputs, rules.FormulaInput{Name: def.Name, Type: string(def.Type)})
	}
	return inputs
}

func ValueInputs(values map[string]any) []rules.FormulaInput {
	inputs := make([]rules.FormulaInput, 0, len(values))
	for name, value := range values {
		inputs = append(inputs, rules.FormulaInput{Name: name, Type: inputType(value)})
	}
	return inputs
}

func inputType(value any) string {
	switch value.(type) {
	case bool:
		return string(formulatypes.VariableValueTypeBoolean)
	case string:
		return string(formulatypes.VariableValueTypeString)
	case nil:
		return string(formulatypes.VariableValueTypeAny)
	default:
		return string(formulatypes.VariableValueTypeNumber)
	}
}

func TemplateSources(template *formulatemplate.FormulaTemplate) []CheckSource {
	sources := make([]CheckSource, 0, len(template.BreakdownDefinitions)+1)
	sources = append(sources, CheckSource{Scope: MainScope, Source: template.Expression})
	for index, def := range template.BreakdownDefinitions {
		if def == nil {
			continue
		}
		sources = append(sources, CheckSource{
			Scope:  fmt.Sprintf("breakdownDefinitions[%d].expression", index),
			Source: def.Expression,
		})
	}
	return sources
}

func (s *Service) CheckTemplate(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) ([]Checked, error) {
	return s.Check(ctx, &CheckRequest{
		SchemaID: template.SchemaID,
		Inputs:   TemplateInputs(template.VariableDefinitions),
		TenantInfo: pagination.TenantInfo{
			OrgID: template.OrganizationID,
			BuID:  template.BusinessUnitID,
		},
		Expect:  rules.ExpectAmount,
		Sources: TemplateSources(template),
	})
}

func Refusal(checked []Checked) error {
	multiErr := errortypes.NewMultiError()
	for _, item := range checked {
		diagnostic := item.Check.FirstError()
		if diagnostic == nil {
			continue
		}
		multiErr.Add(
			item.Scope,
			errortypes.ErrInvalid,
			diagnostic.Message.Template,
			diagnostic.Message.Args...,
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) IsReservedName(ctx context.Context, name string) (bool, error) {
	catalog, err := rules.Catalog(ctx)
	if err != nil {
		return false, err
	}
	return catalog.IsReserved(name), nil
}
