package aitask_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/stretchr/testify/assert"
)

func TestNova_NamesTheTasksTheNovaHarnessRuns(t *testing.T) {
	t.Parallel()

	nova := []aitask.Task{
		aitask.TaskDocumentClassification,
		aitask.TaskDocumentExtraction,
		aitask.TaskBillingDiagnosis,
		aitask.TaskFormulaAssistant,
		aitask.TaskOperationalInsights,
		aitask.TaskDailyBriefing,
		aitask.TaskQueryCompose,
		aitask.TaskChapterTitling,
		aitask.TaskInboundClassification,
		aitask.TaskAccountingMapping,
		aitask.TaskRemittanceExtraction,
		aitask.TaskPaperworkReading,
		aitask.TaskPaletteAnswer,
	}
	for _, task := range nova {
		assert.Truef(t, task.Nova(), "%s runs through the Nova harness", task)
	}
}

func TestRoutedTasks_AreTheDeskTasksAndEmbeddingInOrder(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []aitask.Task{
		aitask.TaskScopeClassification,
		aitask.TaskAssistantChat,
		aitask.TaskEvaluationJudge,
		aitask.TaskEmbedding,
		aitask.TaskGeneral,
	}, aitask.RoutedTasks())
}

func TestBorrowsFrom_OnlyTheScopeGuardBorrowsTheDesksProviders(t *testing.T) {
	t.Parallel()

	for _, task := range aitask.AllTasks() {
		want := aitask.Task("")
		if task == aitask.TaskScopeClassification {
			want = aitask.TaskAssistantChat
		}
		assert.Equalf(t, want, task.BorrowsFrom(), "%s", task)
	}
}
