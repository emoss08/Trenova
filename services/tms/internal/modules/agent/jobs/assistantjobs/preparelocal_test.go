package assistantjobs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recordedEvents(t *testing.T, name string) []map[string]any {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "replay", name))
	require.NoError(t, err)
	var history struct {
		Events []map[string]any `json:"events"`
	}
	require.NoError(t, sonic.Unmarshal(data, &history))

	return history.Events
}

func eventName(event map[string]any) string {
	for key, value := range event {
		attrs, ok := value.(map[string]any)
		if !ok || len(key) < len("Attributes") || key[len(key)-len("Attributes"):] != "Attributes" {
			continue
		}
		if activity, ok := attrs["activityType"].(map[string]any); ok {
			name, _ := activity["name"].(string)
			return name
		}
		if marker, ok := attrs["markerName"].(string); ok {
			return marker
		}
	}

	return ""
}

/*
Preparing a turn ran as a regular activity: a dispatch through the task queue,
then a second workflow task, before the model call could even be scheduled.
Run locally, the turn's first workflow task prepares it and schedules the model
call itself, and the history records the preparation as a marker.
*/
func TestAssistantTurnWorkflow_PreparesLocallyBeforeTheFirstModelCall(t *testing.T) {
	t.Parallel()

	tasks := 0
	for _, event := range recordedEvents(t, "prepared-locally-with-a-tool-read.json") {
		eventType, _ := event["eventType"].(string)
		name := eventName(event)
		assert.NotEqual(t, "PrepareTurnActivity", name, "preparing is not dispatched as an activity")
		if eventType == "EVENT_TYPE_WORKFLOW_TASK_COMPLETED" {
			tasks++
		}
		if eventType == "EVENT_TYPE_ACTIVITY_TASK_SCHEDULED" && name == "ModelCallActivity" {
			assert.Equal(t, 1, tasks, "the first workflow task schedules the model call")
			return
		}
	}
	t.Fatal("the recorded turn never called the model")
}
