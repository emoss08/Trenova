package engine_test

import (
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type abandoningLookup struct {
	release          chan struct{}
	abandonedStarted chan struct{}
	releaseOnce      sync.Once
	startedOnce      sync.Once
}

func newAbandoningLookup() *abandoningLookup {
	return &abandoningLookup{
		release:          make(chan struct{}),
		abandonedStarted: make(chan struct{}),
	}
}

func (l *abandoningLookup) Lookup(table string, _ any) (float64, error) {
	switch table {
	case "slow":
		select {
		case <-l.release:
		case <-time.After(10 * time.Second):
		}
	case "abandoned":
		l.startedOnce.Do(func() { close(l.abandonedStarted) })
	case "fast":
		l.releaseOnce.Do(func() {
			close(l.release)
			select {
			case <-l.abandonedStarted:
			case <-time.After(10 * time.Second):
			}
		})
	}
	return 1, nil
}

func (l *abandoningLookup) Has(string) bool { return true }

func (l *abandoningLookup) Lookup2(string, any, any) (float64, error) { return 1, nil }

func (l *abandoningLookup) Has2(string) bool { return true }

func TestEvaluate_AnAbandonedBreakdownLeavesNoTraceInTheReceipt(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)
	template := &formulatemplate.FormulaTemplate{
		SchemaID:   "test",
		Expression: `lookup("main", 1)`,
		BreakdownDefinitions: []*formulatypes.BreakdownDefinition{
			{
				Name:       "timesOut",
				Label:      "Times out",
				Expression: `lookup("slow", 1) + reduce(1..400, #acc + lookup("abandoned", #), 0)`,
			},
			{
				Name:       "afterwards",
				Label:      "Afterwards",
				Expression: `reduce(1..400, #acc + lookup("fast", #), 0)`,
			},
		},
	}

	ctx := engine.WithEvaluationTimeout(t.Context(), 100*time.Millisecond)
	result, err := e.Evaluate(ctx, &formulatemplatetypes.EvaluationRequest{
		Template: template,
		Entity:   &TestShipment{Weight: 1},
		Lookup:   newAbandoningLookup(),
	})
	require.NoError(t, err)
	require.Len(t, result.Breakdown, 2)
	assert.Contains(t, result.Breakdown[0].Error, "time budget")
	assert.Empty(t, result.Breakdown[1].Error)

	tables := make(map[string]int, 3)
	for _, trace := range result.Receipt.Lookups {
		tables[trace.Table]++
	}
	assert.Equal(t, map[string]int{"main": 1, "fast": 400}, tables,
		"a timed-out evaluation's lookups never reach the receipt")
}
