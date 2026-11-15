package aitask_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The router serves a task only when a feature that pays for it is on, so a
// task that names none could never run and one added without a feature
// fails here first.
func TestEveryTaskNamesTheFeaturesThatPayForIt(t *testing.T) {
	t.Parallel()

	for _, task := range aitask.AllTasks() {
		features := task.Features()
		require.NotEmptyf(t, features, "%s names no plan feature", task)
		for _, feature := range features {
			_, known := platformcatalog.Lookup(feature)
			assert.Truef(t, known, "%s names an unknown feature %s", task, feature)
		}
	}
}

// Reading a document into fields is Document Intelligence, which runs with
// a model provider and without Trenova AI; every other task is Trenova AI.
func TestOnlyReadingDocumentsRunsWithoutTrenovaAI(t *testing.T) {
	t.Parallel()

	documents := []aitask.Task{
		aitask.TaskDocumentClassification,
		aitask.TaskDocumentExtraction,
		aitask.TaskRemittanceExtraction,
		aitask.TaskPaperworkReading,
	}
	for _, task := range aitask.AllTasks() {
		for _, feature := range task.Features() {
			addOn, ok := platformcatalog.AddOnFor(feature)
			require.Truef(t, ok, "%s: %s belongs to no add-on", task, feature)
			want := platformcatalog.AddOnAI
			for _, document := range documents {
				if task == document {
					want = platformcatalog.AddOnDocumentIntelligence
				}
			}
			assert.Equalf(t, want, addOn.Key, "%s", task)
		}
	}
}
