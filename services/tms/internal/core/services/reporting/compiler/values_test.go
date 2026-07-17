package compiler

import (
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gqlNumber mirrors encoding/json.Number (which gqlgen produces for the Any
// and JSON scalars) without importing the lint-forbidden encoding/json.
type gqlNumber string

func (n gqlNumber) Int64() (int64, error)     { return strconv.ParseInt(string(n), 10, 64) }
func (n gqlNumber) Float64() (float64, error) { return strconv.ParseFloat(string(n), 64) }
func (n gqlNumber) String() string            { return string(n) }

func TestCoerceIntegerRepresentations(t *testing.T) {
	for name, value := range map[string]any{
		"int":         int(30),
		"int64":       int64(30),
		"float64":     float64(30),
		"string":      "30",
		"json.Number": gqlNumber("30"),
	} {
		got, err := coerceInteger(value)
		require.NoError(t, err, name)
		assert.EqualValues(t, 30, got, name)
	}

	for name, value := range map[string]any{
		"fractional float":  float64(30.5),
		"fractional number": gqlNumber("30.5"),
		"garbage string":    "thirty",
		"bool":              true,
	} {
		_, err := coerceInteger(value)
		require.Error(t, err, name)
	}
}

func TestCoerceDecimalRepresentations(t *testing.T) {
	for name, value := range map[string]any{
		"float64":     float64(19.75),
		"string":      "19.75",
		"json.Number": gqlNumber("19.75"),
	} {
		got, err := coerceDecimal(value)
		require.NoError(t, err, name)
		assert.Equal(t, "19.75", got.String(), name)
	}

	_, err := coerceDecimal(gqlNumber("not-a-number"))
	require.Error(t, err)
}

func TestCoerceParamValueAcceptsJSONNumber(t *testing.T) {
	param := &report.ParameterDef{
		Name:     "horizonDays",
		Type:     reportcatalog.FieldInt,
		Required: true,
	}

	got, err := coerceParamValue(param, gqlNumber("30"))
	require.NoError(t, err)
	assert.EqualValues(t, int64(30), got)

	multi := &report.ParameterDef{
		Name:  "windows",
		Type:  reportcatalog.FieldInt,
		Multi: true,
	}
	gotMulti, err := coerceParamValue(multi, []any{gqlNumber("7"), gqlNumber("30")})
	require.NoError(t, err)
	assert.Equal(t, []any{int64(7), int64(30)}, gotMulti)
}

// TestCompileParamsFromGraphQLNumbers reproduces the reported failure:
// "parameter \"horizonDays\": expected an integer, got json.Number" — params
// bound through the GraphQL JSON scalar must compile.
func TestCompileParamsFromGraphQLNumbers(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	def := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
		Filters: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{
				{
					Ref:      report.FieldRef{Field: "createdAt"},
					Operator: "lastndays",
					Param:    "horizonDays",
				},
			},
		},
		Parameters: []report.ParameterDef{
			{Name: "horizonDays", Type: reportcatalog.FieldInt, Required: true},
		},
	}

	req := newRequest(def)
	req.Params = map[string]any{"horizonDays": gqlNumber("30")}

	compiled, err := c.Compile(t.Context(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, compiled.SQL)
}

func TestRelativePeriodOperators(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	c := newTestCompiler(allowAllEngine())

	compileWith := func(op dbtype.Operator) *services.CompiledReportQuery {
		compiled, cErr := c.Compile(t.Context(), newRequest(&report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
			Filters: &report.FilterGroup{
				Op:      report.BoolOpAnd,
				Filters: []report.FieldFilter{{Ref: report.FieldRef{Field: "createdAt"}, Operator: op}},
			},
		}))
		require.NoError(t, cErr, op)
		return compiled
	}

	rangeBinds := func(compiled *services.CompiledReportQuery) (start, end int64) {
		var bounds []int64
		for _, arg := range compiled.Args {
			if v, ok := arg.(int64); ok {
				bounds = append(bounds, v)
			}
		}
		require.Len(t, bounds, 2)
		return bounds[0], bounds[1]
	}

	// testNowUnix = 2026-07-15T12:00:00Z → 2026-07-15 in America/Chicago.
	t.Run("thismonth", func(t *testing.T) {
		start, end := rangeBinds(compileWith(dbtype.OpThisMonth))
		assert.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, chicago).Unix(), start)
		assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, chicago).Unix(), end)
	})

	t.Run("lastquarter", func(t *testing.T) {
		start, end := rangeBinds(compileWith(dbtype.OpLastQuarter))
		assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, chicago).Unix(), start)
		assert.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, chicago).Unix(), end)
	})

	t.Run("lastyear", func(t *testing.T) {
		start, end := rangeBinds(compileWith(dbtype.OpLastYear))
		assert.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, chicago).Unix(), start)
		assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, chicago).Unix(), end)
	})

	t.Run("thisweek", func(t *testing.T) {
		start, end := rangeBinds(compileWith(dbtype.OpThisWeek))
		startDay := time.Unix(start, 0).In(chicago)
		assert.Equal(t, time.Monday, startDay.Weekday())
		assert.Equal(t, int64(7*86_400), end-start)
		ref := time.Date(2026, 7, 15, 12, 0, 0, 0, chicago).Unix()
		assert.LessOrEqual(t, start, ref)
		assert.Greater(t, end, ref)
	})

	t.Run("lastweek", func(t *testing.T) {
		thisStart, _ := rangeBinds(compileWith(dbtype.OpThisWeek))
		lastStart, lastEnd := rangeBinds(compileWith(dbtype.OpLastWeek))
		assert.Equal(t, thisStart, lastEnd)
		assert.Equal(t, int64(7*86_400), lastEnd-lastStart)
	})

	t.Run("rejected on non-epoch fields", func(t *testing.T) {
		_, cErr := c.Compile(t.Context(), newRequest(&report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
			Filters: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{Ref: report.FieldRef{Field: "proNumber"}, Operator: dbtype.OpThisMonth},
				},
			},
		}))
		require.Error(t, cErr)
	})
}

func TestParamAllowedValues(t *testing.T) {
	param := &report.ParameterDef{
		Name:          "statusGroup",
		Type:          reportcatalog.FieldString,
		AllowedValues: []string{"active", "inactive"},
	}

	got, err := coerceParamValue(param, "active")
	require.NoError(t, err)
	assert.Equal(t, "active", got)

	_, err = coerceParamValue(param, "archived")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowed values")

	multi := &report.ParameterDef{
		Name:          "windows",
		Type:          reportcatalog.FieldInt,
		Multi:         true,
		AllowedValues: []string{"7", "30", "90"},
	}
	gotMulti, err := coerceParamValue(multi, []any{gqlNumber("7"), gqlNumber("30")})
	require.NoError(t, err)
	assert.Equal(t, []any{int64(7), int64(30)}, gotMulti)

	_, err = coerceParamValue(multi, []any{gqlNumber("14")})
	require.Error(t, err)
}
