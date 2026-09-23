package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newAssistantForTest(t *testing.T) (*Assistant, *prometheus.Registry) {
	t.Helper()

	registry := prometheus.NewRegistry()

	return NewAssistant(registry, zap.NewNop(), true), registry
}

func TestAssistant_CountsTurnsByHowTheyEnded(t *testing.T) {
	t.Parallel()

	assistant, registry := newAssistantForTest(t)

	assistant.RecordTurn("Completed", 3)
	assistant.RecordTurn("Completed", 4)
	assistant.RecordTurn("Failed", 1)

	assert.Equal(t, 2, countOf(t, registry, "trenova_assistant_turns_total", "Completed"))
	assert.Equal(t, 1, countOf(t, registry, "trenova_assistant_turns_total", "Failed"))
}

// A turn that never said anything is not one that answered instantly. Only a
// reader who saw the first event records how long it took.
func TestAssistant_RecordsFirstEventsOnlyWhenOneArrived(t *testing.T) {
	t.Parallel()

	assistant, registry := newAssistantForTest(t)

	assistant.RecordTurn("Failed", 5)
	assert.Zero(t, firstEventSamples(t, registry),
		"finishing a turn says nothing about when it first spoke")

	assistant.RecordFirstEvent(0.8)
	assert.Equal(t, uint64(1), firstEventSamples(t, registry))
}

func firstEventSamples(t *testing.T, registry *prometheus.Registry) uint64 {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != "trenova_assistant_turn_first_event_seconds" {
			continue
		}
		for _, metric := range family.GetMetric() {
			return metric.GetHistogram().GetSampleCount()
		}
	}

	return 0
}

// A stop that silently cancelled nothing is the failure mode worth seeing:
// the reader is told it stopped while the turn runs on and bills.
func TestAssistant_SeparatesAStopThatCancelledFromOneThatDidNot(t *testing.T) {
	t.Parallel()

	assistant, registry := newAssistantForTest(t)

	assistant.RecordTurnStopped("cancelled")
	assistant.RecordTurnStopped("already_ended")

	assert.Equal(t, 1, countOf(t, registry, "trenova_assistant_turns_stopped_total", "cancelled"))
	assert.Equal(t, 1,
		countOf(t, registry, "trenova_assistant_turns_stopped_total", "already_ended"))
}

// A collector is something a caller may legitimately not have: a test builds
// its service without one. Measuring must never be a reason to crash.
func TestAssistant_IsSafeWhenThereIsNoCollector(t *testing.T) {
	t.Parallel()

	var absent *Assistant

	assert.NotPanics(t, func() {
		absent.RecordTurn("Completed", 1)
		absent.RecordFirstEvent(1)
		absent.RecordTurnStopped("cancelled")
		absent.RecordStepReplayed("AssistantTurn", "Completed")
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
