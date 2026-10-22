package formula

import (
	"context"
	"slices"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

const MainScope = "expression"

type evaluation struct {
	tenant          pagination.TenantInfo
	env             map[string]any
	expression      string
	breakdowns      []*formulatypes.BreakdownDefinition
	traceStatements bool
}

type BreakdownLine struct {
	Index int
	Name  string
	Label string
	Value decimal.Decimal
	Error *rules.FormulaError
}

type Evaluated struct {
	Amount     decimal.Decimal
	Error      *rules.FormulaError
	Lines      []BreakdownLine
	Statements []formulatemplatetypes.Statement
	Lookups    []formulatypes.LookupTrace
	Stamp      string
}

func (e *Evaluated) Breakdown(locale i18n.Locale) []formulatemplatetypes.BreakdownAmount {
	if len(e.Lines) == 0 {
		return nil
	}
	amounts := make([]formulatemplatetypes.BreakdownAmount, 0, len(e.Lines))
	for _, line := range e.Lines {
		item := formulatemplatetypes.BreakdownAmount{Name: line.Name, Label: line.Label}
		if line.Error != nil {
			item.Error = line.Error.Localize(locale)
		} else {
			item.Amount = line.Value
		}
		amounts = append(amounts, item)
	}
	return amounts
}

func (s *Service) evaluate(ctx context.Context, ev *evaluation) (*Evaluated, error) {
	sources := make([]string, 0, len(ev.breakdowns)+1)
	sources = append(sources, ev.expression)
	programs := make([]rules.FormulaProgram, 0, len(ev.breakdowns)+1)
	programs = append(programs, rules.FormulaProgram{
		ID:     MainScope,
		Source: ev.expression,
		Expect: rules.ExpectAmount,
	})
	lines := make([]BreakdownLine, 0, len(ev.breakdowns))
	for index, def := range ev.breakdowns {
		if def == nil {
			continue
		}
		sources = append(sources, def.Expression)
		programs = append(programs, rules.FormulaProgram{
			ID:     strconv.Itoa(index),
			Source: def.Expression,
			Expect: rules.ExpectAmount,
		})
		lines = append(lines, BreakdownLine{Index: index, Name: def.Name, Label: def.Label})
	}

	selection, err := s.tablesFor(ctx, ev.tenant, sources...)
	if err != nil {
		return nil, err
	}

	req := &rules.EvaluateRequest{
		Programs:        programs,
		Env:             ev.env,
		TraceLookups:    true,
		TraceStatements: ev.traceStatements,
	}
	if selection != nil {
		req.Tables = selection.Encoded
	}
	resp, err := rules.EvaluateFormulas(ctx, req)
	if err != nil {
		return nil, err
	}

	evaluated := &Evaluated{Lines: lines}
	if selection != nil {
		evaluated.Stamp = selection.Stamp
	}

	main := &resp.Results[0]
	if amount, amountErr := main.Amount(); amountErr != nil {
		if main.Error == nil {
			return nil, amountErr
		}
		evaluated.Error = main.Error
	} else {
		evaluated.Amount = amount
	}
	evaluated.Statements = main.Statements
	evaluated.Lookups = appendLookups(evaluated.Lookups, MainScope, main.Lookups)

	for index := range evaluated.Lines {
		line := &evaluated.Lines[index]
		result := &resp.Results[index+1]
		evaluated.Lookups = appendLookups(evaluated.Lookups, line.Name, result.Lookups)
		amount, amountErr := result.Amount()
		if amountErr == nil {
			line.Value = amount
			continue
		}
		if result.Error == nil {
			return nil, amountErr
		}
		line.Error = result.Error
	}

	return evaluated, nil
}

func appendLookups(
	traces []formulatypes.LookupTrace,
	scope string,
	lookups []rules.FormulaLookup,
) []formulatypes.LookupTrace {
	for index := range lookups {
		lookup := &lookups[index]
		trace := formulatypes.LookupTrace{
			Scope:    scope,
			Function: lookup.Function,
			Table:    lookup.Table,
			Keys:     traceKeys(lookup.Keys),
		}
		if lookup.Value != nil {
			trace.Value = formulatypes.Number{Decimal: *lookup.Value}
		}
		if lookup.Match != nil {
			trace.Match = &formulatypes.LookupMatch{
				MatchedKey: lookup.Match.MatchedKey,
				BandMin:    lookup.Match.BandMin,
				BandMax:    lookup.Match.BandMax,
				Adjusted:   lookup.Match.Adjusted,
			}
		}
		if lookup.Error != nil {
			trace.Error = lookup.Error.Error()
		}
		traces = append(traces, trace)
	}
	return traces
}

func traceKeys(keys []rules.RawJSON) []any {
	values := make([]any, 0, len(keys))
	for _, key := range keys {
		var value any
		if err := sonic.Unmarshal(key, &value); err != nil {
			value = string(key)
		}
		values = append(values, value)
	}
	return values
}

func (e *Evaluated) Receipt(variables []formulatypes.VariableProvenance) *formulatypes.Receipt {
	result := &formulatypes.Receipt{
		Variables: variables,
		Lookups:   e.Lookups,
		RawAmount: e.Amount,
	}
	if len(result.Lookups) > 0 {
		result.RateTableStamp = e.Stamp
	}
	return result
}

func fieldsIn(fields []string, failures map[string]error) []string {
	matched := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, failed := failures[field]; failed && !slices.Contains(matched, field) {
			matched = append(matched, field)
		}
	}
	return matched
}
