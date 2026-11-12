package aitask

// Task names a unit of AI work. A Desk task (the chat, its scope guard, quality
// judging) and embedding are routed to whichever providers an organization
// assigns them, since the Desk lets an organization pick its model. Nova's
// tasks (see Nova) all run on the organization's Nova provider with Nova's own
// model; their names say what the usage was for.
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
	// produces a number, a severity or a link: those are computed before it runs
	// and checked after it.
	TaskOperationalInsights = Task("OperationalInsights")
	// TaskDailyBriefing writes the morning page from figures already
	// gathered. Like the insight narration it produces only wording, and
	// every number it writes is checked against those figures afterwards.
	TaskDailyBriefing = Task("DailyBriefing")
	// TaskQueryCompose turns a sentence into a table's filters. It names
	// fields and values from a catalogue it is shown and writes no SQL;
	// everything it names is compiled against that catalogue afterwards, so a
	// wrong guess produces an unresolved line rather than a wrong answer.
	TaskQueryCompose = Task("QueryCompose")
	// TaskChapterTitling decides whether a Desk question starts a new
	// chapter and names it in a few words. It runs once per question on a
	// short prompt and its answer is cleaned and checked against the
	// conversation afterwards.
	TaskChapterTitling = Task("ChapterTitling")
	// TaskInboundClassification decides what an email that arrived actually is.
	// Like the other classifiers it answers with a label from a fixed set, and
	// the label is re-checked against that set afterwards: a model that invents
	// a category produces a message for a person, not a new kind of work.
	TaskInboundClassification = Task("InboundClassification")
	TaskAccountingMapping     = Task("AccountingMapping")
	// TaskRemittanceExtraction reads the lines of a factor's PDF remittance.
	// It is kept apart from DocumentExtraction so its usage and its training
	// data are its own: a remittance is not the kind of document extraction
	// reads. What it reads is matched to schedule items and a person applies
	// it, so it never books anything on its own.
	TaskRemittanceExtraction = Task("RemittanceExtraction")
	// TaskPaperworkReading reads a shipment's billing paperwork (a proof of
	// delivery, a bill of lading, a receipt) for the facts billing checks, in a
	// single structured call that reads the file itself, without waiting for
	// the document to be converted.
	TaskPaperworkReading = Task("PaperworkReading")
	// TaskPaletteAnswer answers a quick question typed into the command
	// palette: a short streamed answer from a few read-only lookups, run
	// through the Nova harness rather than as a Desk turn.
	TaskPaletteAnswer = Task("PaletteAnswer")
	// TaskCaseReading writes Nova's reading of a record's case: the verdict
	// a person reads first, citing the numbered evidence beside it. It
	// produces wording only, and every number it writes is checked against
	// that evidence afterwards.
	TaskCaseReading     = Task("CaseReading")
	TaskEvaluationJudge = Task("EvaluationJudge")
	TaskEmbedding       = Task("Embedding")
	TaskGeneral         = Task("General")
)

// BorrowsFrom is the task whose providers serve this one after its own: the
// scope classifier falls back to the model answering the Desk, so a guard that
// no provider was assigned to, or whose providers are down, still runs.
func (t Task) BorrowsFrom() Task {
	if t == TaskScopeClassification {
		return TaskAssistantChat
	}

	return ""
}

// Nova reports a task Nova does through its harness. Nova's tasks are never
// assigned to providers: each runs on the organization's Nova provider with
// aiprovider.NovaModel, so an organization chooses only whose key Nova uses.
// The Desk's tasks and embeddings are routed by assignment.
func (t Task) Nova() bool {
	switch t {
	case TaskDocumentClassification,
		TaskDocumentExtraction,
		TaskBillingDiagnosis,
		TaskFormulaAssistant,
		TaskOperationalInsights,
		TaskDailyBriefing,
		TaskQueryCompose,
		TaskChapterTitling,
		TaskInboundClassification,
		TaskAccountingMapping,
		TaskRemittanceExtraction,
		TaskPaperworkReading,
		TaskPaletteAnswer,
		TaskCaseReading:
		return true
	case TaskScopeClassification,
		TaskAssistantChat,
		TaskEvaluationJudge,
		TaskEmbedding,
		TaskGeneral:
		return false
	default:
		return false
	}
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
		TaskPaperworkReading,
		TaskPaletteAnswer,
		TaskCaseReading,
		TaskEvaluationJudge,
		TaskEmbedding,
		TaskGeneral:
		return true
	default:
		return false
	}
}

// AllTasks lists every task, Nova's included, for reading usage by task.
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
		TaskPaperworkReading,
		TaskPaletteAnswer,
		TaskCaseReading,
		TaskEvaluationJudge,
		TaskEmbedding,
		TaskGeneral,
	}
}

// RoutedTasks lists the tasks an organization assigns to its providers, for
// building configuration UIs and route previews: every task but Nova's.
func RoutedTasks() []Task {
	all := AllTasks()
	routed := make([]Task, 0, len(all))
	for _, task := range all {
		if !task.Nova() {
			routed = append(routed, task)
		}
	}

	return routed
}

func (t Task) Generates() bool {
	return t != TaskEmbedding
}
