package aitask

// Task names a unit of AI work that can be routed to a provider independently.
// Splitting them is the point of the abstraction: classification is high-volume
// and cheap enough for a local model, while a billing diagnosis writes to the
// ledger and warrants a frontier model.
type Task string

const (
	TaskDocumentClassification = Task("DocumentClassification")
	TaskDocumentExtraction     = Task("DocumentExtraction")
	TaskBillingDiagnosis       = Task("BillingDiagnosis")
	TaskFormulaAssistant       = Task("FormulaAssistant")
	// TaskScopeClassification decides whether a request belongs to this system at
	// all. It runs on every chat turn and needs only a yes/no with a category, so
	// it is the cheapest thing to route to a small local model.
	TaskScopeClassification = Task("ScopeClassification")
	// TaskAssistantChat answers a person's question and drives the tool loop.
	TaskAssistantChat = Task("AssistantChat")
	// TaskOperationalInsights writes the prose on a home-screen insight. It never
	// produces a number, a severity or a link — those are computed before it runs
	// and checked after it — so it is safe to route to whatever is cheapest.
	TaskOperationalInsights = Task("OperationalInsights")
	// TaskDailyBriefing writes the morning page from figures already
	// gathered. Like the insight narration it produces only wording, and
	// every number it writes is checked against those figures afterwards,
	// so it is safe to route wherever is cheapest.
	TaskDailyBriefing = Task("DailyBriefing")
	// TaskQueryCompose turns a sentence into a table's filters. It names
	// fields and values from a catalogue it is shown and writes no SQL;
	// everything it names is compiled against that catalogue afterwards, so a
	// cheap model that guesses wrong produces an unresolved line rather than a
	// wrong answer.
	TaskQueryCompose = Task("QueryCompose")
	// TaskChapterTitling decides whether a Desk question starts a new
	// chapter and names it in a few words. It runs once per question on a
	// short prompt and its answer is cleaned and checked against the
	// conversation afterwards, so the cheapest fast model is the right one.
	TaskChapterTitling = Task("ChapterTitling")
	// TaskInboundClassification decides what an email that arrived actually is.
	// Like the other classifiers it answers with a label from a fixed set, and
	// the label is re-checked against that set afterwards — a model that invents
	// a category produces a message for a person, not a new kind of work.
	TaskInboundClassification = Task("InboundClassification")
	TaskAccountingMapping     = Task("AccountingMapping")
	// TaskRemittanceExtraction reads the lines of a factor's PDF remittance.
	// It is kept apart from DocumentExtraction because the fine-tuned
	// extraction model, its shadow runs and its training data all key on that
	// task, and a remittance is not the kind of document they were built for.
	// What it reads is matched to schedule items and a person applies it, so
	// it never books anything on its own.
	TaskRemittanceExtraction = Task("RemittanceExtraction")
	TaskEvaluationJudge      = Task("EvaluationJudge")
	TaskEmbedding            = Task("Embedding")
	TaskGeneral              = Task("General")
)

// BorrowsFrom is the task whose providers serve this one after its own: the
// scope classifier falls back to the model answering the Desk, so a guard that
// no provider was assigned to, or whose providers are down, still runs.
func (t Task) BorrowsFrom() Task {
	if t == TaskScopeClassification || t == TaskChapterTitling {
		return TaskAssistantChat
	}

	return ""
}

func (t Task) IsValid() bool {
	switch t {
	case TaskDocumentClassification,
		TaskDocumentExtraction,
		TaskBillingDiagnosis,
		TaskFormulaAssistant,
		TaskScopeClassification,
		TaskAssistantChat,
		TaskOperationalInsights,
		TaskDailyBriefing,
		TaskQueryCompose,
		TaskChapterTitling,
		TaskInboundClassification,
		TaskAccountingMapping,
		TaskRemittanceExtraction,
		TaskEvaluationJudge,
		TaskEmbedding,
		TaskGeneral:
		return true
	default:
		return false
	}
}

// AllTasks lists every routable task, for building configuration UIs.
func AllTasks() []Task {
	return []Task{
		TaskDocumentClassification,
		TaskDocumentExtraction,
		TaskBillingDiagnosis,
		TaskFormulaAssistant,
		TaskScopeClassification,
		TaskAssistantChat,
		TaskOperationalInsights,
		TaskDailyBriefing,
		TaskQueryCompose,
		TaskChapterTitling,
		TaskInboundClassification,
		TaskAccountingMapping,
		TaskRemittanceExtraction,
		TaskEvaluationJudge,
		TaskEmbedding,
		TaskGeneral,
	}
}

// WritesToLedger reports whether a task's output can drive a financial mutation.
// Such a task is refused a provider that has not been marked trusted, so a
// misconfigured 7B model cannot quietly propose ledger corrections.
func (t Task) WritesToLedger() bool {
	return t == TaskBillingDiagnosis
}

func (t Task) Generates() bool {
	return t != TaskEmbedding
}
