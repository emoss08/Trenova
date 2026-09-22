package metrics

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newAssistantForTest(t *testing.T) (*Assistant, *prometheus.Registry) {
	t.Helper()

	registry := prometheus.NewRegistry()

	return NewAssistant(registry, zap.NewNop(), true), registry
}

// The label is the point. The issue this work comes from asks for the durable
// runtime to be compared against the one it replaces, and a figure that does
// not say which runtime produced it cannot answer that.
func TestAssistant_KeepsTheTwoRuntimesApart(t *testing.T) {
	t.Parallel()

	assistant, registry := newAssistantForTest(t)

	assistant.RecordTurn(TransportInProcess, "Completed", 3, 1)
	assistant.RecordTurn(TransportDurable, "Completed", 4, 2)
	assistant.RecordTurn(TransportDurable, "Failed", 1, -1)

	assert.Equal(t, 1, countOf(t, registry,
		"trenova_assistant_turns_total", TransportInProcess, "Completed"))
	assert.Equal(t, 1, countOf(t, registry,
		"trenova_assistant_turns_total", TransportDurable, "Completed"))
	assert.Equal(t, 1, countOf(t, registry,
		"trenova_assistant_turns_total", TransportDurable, "Failed"))
}

// A turn that never said anything is not one that answered instantly, and a
// histogram that recorded it as zero would report the worst case as the best.
func TestAssistant_DoesNotCountASilentTurnAsInstant(t *testing.T) {
	t.Parallel()

	assistant, registry := newAssistantForTest(t)

	assistant.RecordTurn(TransportDurable, "Failed", 5, -1)

	assert.Equal(t, 0,
		testutil.CollectAndCount(registry, "trenova_assistant_turn_first_event_seconds"),
		"a turn that never spoke contributes no first-event observation")
}

// A stop that silently cancelled nothing is the failure mode worth seeing:
// the reader is told it stopped while the turn runs on and bills.
func TestAssistant_SeparatesAStopThatCancelledFromOneThatDidNot(t *testing.T) {
	t.Parallel()

	assistant, registry := newAssistantForTest(t)

	assistant.RecordTurnStopped(TransportDurable, "cancelled")
	assistant.RecordTurnStopped(TransportInProcess, "no_execution")

	assert.Equal(t, 1, countOf(t, registry,
		"trenova_assistant_turns_stopped_total", TransportDurable, "cancelled"))
	assert.Equal(t, 1, countOf(t, registry,
		"trenova_assistant_turns_stopped_total", TransportInProcess, "no_execution"))
}

func TestAssistant_CountsBytesOnlyForAPublishThatLanded(t *testing.T) {
	t.Parallel()

	assistant, registry := newAssistantForTest(t)

	assistant.RecordStreamPublish("delta", 120, nil)
	assistant.RecordStreamPublish("delta", 500, errors.New("connection refused"))

	assert.Equal(t, 1, countOf(t, registry, "trenova_assistant_stream_events_total", "delta", "success"))
	assert.Equal(t, 1, countOf(t, registry, "trenova_assistant_stream_events_total", "delta", "error"))
	assert.InDelta(t, 120.0, gaugeOf(t, registry, "trenova_assistant_stream_bytes_total"), 0.01,
		"bytes that never reached redis were not published")
}

// A collector is something a caller may legitimately not have: a test builds
// its service without one. Measuring must never be a reason to crash.
func TestAssistant_IsSafeWhenThereIsNoCollector(t *testing.T) {
	t.Parallel()

	var absent *Assistant

	assert.NotPanics(t, func() {
		absent.RecordTurn(TransportDurable, "Completed", 1, 1)
		absent.RecordFirstEvent(TransportDurable, 1)
		absent.RecordTurnStopped(TransportDurable, "cancelled")
		absent.RecordStepReplayed("AssistantTurn", "Completed")
		absent.RecordStreamPublish("delta", 10, nil)
		absent.RecordStreamAttach("resumed")
	})
}

// countOf reads one counter by the label values it carries, in any order:
// prometheus sorts labels alphabetically when it gathers, so matching them
// positionally would pass or fail on the declaration order of the metric
// rather than on what was recorded.
func countOf(t *testing.T, registry *prometheus.Registry, name string, labels ...string) int {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			values := make([]string, 0, len(metric.GetLabel()))
			for _, label := range metric.GetLabel() {
				values = append(values, label.GetValue())
			}
			if carriesAll(values, labels) {
				return int(metric.GetCounter().GetValue())
			}
		}
	}

	return 0
}

func gaugeOf(t *testing.T, registry *prometheus.Registry, name string) float64 {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)

	total := 0.0
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			total += metric.GetCounter().GetValue()
		}
	}

	return total
}

func carriesAll(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for _, value := range want {
		found := false
		for _, candidate := range got {
			if candidate == value {
				found = true

				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
