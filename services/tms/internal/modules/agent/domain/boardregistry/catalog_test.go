package boardregistry_test

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homeboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	shipment   = recordkind.Kind(permission.ResourceShipment)
	credential = recordkind.Kind(permission.ResourceWorkerCredential)
)

func TestMeasureOfReadsANameOrAnAliasInAnyCase(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"lateness", "Lateness", "lateBy", "LATEMINUTES"} {
		measure, ok := boardregistry.Catalog().MeasureOf(shipment, name)
		require.True(t, ok, name)
		assert.Equal(t, "lateness", measure.Name)
	}
	revenue, ok := boardregistry.Catalog().MeasureOf(shipment, "totalCharge")
	require.True(t, ok)
	assert.Equal(t, "revenue", revenue.Name)

	expiry, ok := boardregistry.Catalog().MeasureOf(credential, "expiresAt")
	require.True(t, ok)
	assert.Equal(t, "expiry", expiry.Name)
	assert.Equal(t, boardcatalog.ShapeCountdown, expiry.Shape)
	assert.True(t, expiry.Ascending)
	assert.False(t, expiry.Summable)

	_, ok = boardregistry.Catalog().MeasureOf(shipment, "expiresAt")
	assert.False(t, ok, "an alias belongs to its own kind")
	_, ok = boardregistry.Catalog().MeasureOf(shipment, "")
	assert.False(t, ok)
}

func TestEveryKindAndMeasureIsWhole(t *testing.T) {
	t.Parallel()

	seen := map[recordkind.Kind]bool{}
	for _, kind := range boardregistry.Catalog().Kinds() {
		assert.False(t, seen[kind.Kind], "%s listed twice", kind.Kind)
		seen[kind.Kind] = true
		assert.True(t, kind.Kind.Known(), "%s has no id prefix", kind.Kind)
		assert.NotEmpty(t, kind.Plural())
	}
	names := map[string]bool{}
	for _, measure := range boardregistry.Catalog().Measures() {
		key := string(measure.Kind) + ":" + measure.Name
		assert.False(t, names[key], "%s listed twice", key)
		names[key] = true
		assert.True(
			t,
			boardregistry.Catalog().Supports(measure.Kind),
			"%s measures a kind the catalog lacks",
			key,
		)
		assert.NotEmpty(t, measure.Shape, key)
		assert.NotEmpty(t, measure.Value, key)
		assert.NotEmpty(t, measure.Label, key)
		assert.NotEmpty(t, measure.Largest, key)
		assert.NotEmpty(t, measure.Heading, key)
		assert.LessOrEqual(t, utf8.RuneCountInString(measure.Words), 60, key)
		if measure.Value == boardcatalog.ValueDuration &&
			measure.Shape != boardcatalog.ShapeAmount && measure.Shape != boardcatalog.ShapeLimit &&
			measure.Shape != boardcatalog.ShapeSpread {
			assert.False(t, measure.Summable, "%s: time a record is behind does not add up", key)
		}
	}
}

var rowedForms = []boardcatalog.Form{
	boardcatalog.FormSpans, boardcatalog.FormGrid, boardcatalog.FormLog, boardcatalog.FormChecks,
	boardcatalog.FormTable,
}

func TestEveryMetricIsWhole(t *testing.T) {
	t.Parallel()

	for _, metric := range boardregistry.Catalog().Metrics() {
		assert.Contains(t, boardcatalog.AllForms(), metric.Form, metric.Name)
		assert.LessOrEqual(t, utf8.RuneCountInString(metric.Words), 120, metric.Name)
		if slices.Contains(rowedForms, metric.Form) {
			assert.Positive(t, metric.Sketch.Rows,
				"%s lists rows, so it reserves them before it is read", metric.Name)
		}
		if metric.For != "" {
			assert.True(t, metric.For.Known(), "%s reads a kind with no id prefix", metric.Name)
		}
		for _, period := range metric.Periods {
			assert.Contains(t, boardcatalog.AllPeriods(), period, metric.Name)
		}
		found, ok := boardregistry.Catalog().MetricNamed(metric.Name)
		require.True(t, ok)
		assert.Equal(t, metric.Name, found.Name)
	}
}

func TestTheHomeFiguresAreTheHomePagesOwn(t *testing.T) {
	t.Parallel()

	for _, name := range []string{homeboard.MetricDispatchToday, billingboard.MetricReceivables} {
		metric, ok := boardregistry.Catalog().MetricNamed(name)
		require.True(t, ok)
		for _, figure := range metric.Figures {
			definition, known := homelayout.MetricDefinitionFor(figure.Name)
			require.True(t, known, "%s is not a home page figure", figure.Name)
			assert.Equal(t, definition.Feature, figure.Feature, figure.Name)
			assert.Contains(t, metric.Requires, definition.Resource, figure.Name)
		}
	}
}

func TestFigureNamedFindsTheOneMetricThatDeclaresIt(t *testing.T) {
	t.Parallel()

	metric, figure, ok := boardregistry.Catalog().FigureNamed("onTimePercent")
	require.True(t, ok)
	assert.Equal(t, homeboard.MetricDispatchToday, metric.Name)
	assert.Equal(t, boardcatalog.ValuePercent, figure.Value)

	_, _, ok = boardregistry.Catalog().FigureNamed("nothing")
	assert.False(t, ok)

	alias, ok := boardregistry.Catalog().MetricNamed("AR")
	require.True(t, ok)
	assert.Equal(t, billingboard.MetricReceivables, alias.Name)
}

func TestAMetricsRecordsAreAKindAndMeasureTheCatalogLists(t *testing.T) {
	t.Parallel()

	catalog := boardregistry.Catalog()
	for _, metric := range catalog.Metrics() {
		if metric.Records == nil {
			continue
		}
		assert.True(t, catalog.Supports(metric.Records.Kind), "%s lists records of %s",
			metric.Name, metric.Records.Kind)
		measure, ok := catalog.MeasureOf(metric.Records.Kind, metric.Records.Measure)
		assert.True(t, ok && measure.Name == metric.Records.Measure,
			"%s measures its records by %s, which %s does not have", metric.Name,
			metric.Records.Measure, metric.Records.Kind)
		assert.NotEmpty(t, metric.Records.Words, metric.Name)
		assert.NotEmpty(t, metric.Records.From, metric.Name)
	}
}

func TestAMetricsLeadsAreItsOwnFiguresAndFitOneAnswer(t *testing.T) {
	t.Parallel()

	for _, metric := range boardregistry.Catalog().Metrics() {
		if len(metric.Leads) == 0 {
			continue
		}
		assert.LessOrEqual(t, len(metric.Leads), board.MaxReadings, metric.Name)
		for _, name := range metric.Leads {
			_, ok := metric.Figure(name)
			assert.True(t, ok, "%s leads with %s, which it does not declare", metric.Name, name)
		}
		leads, rest := metric.Led()
		assert.Len(t, leads, len(metric.Leads), metric.Name)
		assert.Len(t, rest, len(metric.Figures)-len(metric.Leads), metric.Name)
	}
}

type lookupSpace struct {
	t      *testing.T
	name   string
	holder map[string]string
}

func newLookupSpace(t *testing.T, name string) *lookupSpace {
	return &lookupSpace{t: t, name: name, holder: make(map[string]string, 64)}
}

func (s *lookupSpace) claim(word, owner string) {
	s.t.Helper()
	key := strings.ToLower(strings.TrimSpace(word))
	if held, taken := s.holder[key]; taken && held != owner {
		s.t.Errorf("%s: %q names both %s and %s", s.name, word, held, owner)
		return
	}
	s.holder[key] = owner
}

func TestNoNameOrAliasPointsTwoWays(t *testing.T) {
	t.Parallel()

	catalog := boardregistry.Catalog()
	for _, kind := range catalog.Kinds() {
		measures := newLookupSpace(t, "measures of "+string(kind.Kind))
		for _, measure := range catalog.MeasuresOf(kind.Kind) {
			owner := "measure " + measure.Name
			measures.claim(measure.Name, owner)
			for _, alias := range measure.Aliases {
				measures.claim(alias, owner)
			}
		}
	}

	metrics := newLookupSpace(t, "metrics and bare figures")
	for _, metric := range catalog.Metrics() {
		owner := "metric " + metric.Name
		metrics.claim(metric.Name, owner)
		for _, alias := range metric.Aliases {
			metrics.claim(alias, owner)
		}
		figures := newLookupSpace(t, "figures of "+metric.Name)
		for _, figure := range metric.Figures {
			figures.claim(figure.Name, "figure "+figure.Name)
		}
	}
	for _, metric := range catalog.Metrics() {
		for _, figure := range metric.Figures {
			metrics.claim(figure.Name, "figure "+figure.Name+" of "+metric.Name)
		}
	}
}

func TestNoTwoMeasuresOfAKindSayTheSameWord(t *testing.T) {
	t.Parallel()

	catalog := boardregistry.Catalog()
	for _, kind := range catalog.Kinds() {
		said := make(map[string]string, 8)
		for _, measure := range catalog.MeasuresOf(kind.Kind) {
			for _, word := range measure.Says {
				lowered := strings.ToLower(word)
				assert.NotContains(t, said, lowered, "%s and %s of %s both say %q",
					said[lowered], measure.Name, kind.Kind, word)
				said[lowered] = measure.Name
			}
		}
	}
}
