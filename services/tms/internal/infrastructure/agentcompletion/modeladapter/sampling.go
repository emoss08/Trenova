package modeladapter

import "github.com/emoss08/trenova/internal/core/domain/aiprovider"

// Sampling is how adventurously a model picks its next token.
//
// Both are pointers because "unset" and "zero" are different instructions:
// a temperature of 0 asks for the most likely token every time, while an
// absent temperature leaves the endpoint's own default in place. Nothing
// used to be sent at all, which meant every call ran at whatever the
// server chose — typically temperature 1 with no nucleus cutoff, which is
// sampling from the entire vocabulary including its tail. A large model
// mostly survives that. A mixture-of-experts model with a few billion
// active parameters does not: one improbable token is drawn, the context
// is now off-distribution, and the reply degenerates into fragments of
// other languages and invented markup it never recovers from.
type Sampling struct {
	Temperature *float64
	TopP        *float64
}

// SamplingForTask is what a unit of work should be sampled at.
//
// The split is between work that has one right answer and work that is
// wording. A turn that drives tools has to emit an exact tool name and
// exact JSON, and creativity there is only ever a defect; a narration is
// judged on how it reads and its figures are checked afterwards anyway.
// The nucleus cutoff is the same everywhere, because the tail it removes
// is never the right token in either kind of work.
func SamplingForTask(task aiprovider.Task) Sampling {
	temperature := 0.3
	switch task {
	case aiprovider.TaskScopeClassification,
		aiprovider.TaskDocumentClassification,
		aiprovider.TaskInboundClassification,
		// Compiling a sentence into a table's filters is the same kind of work:
		// the answer is a structure checked against a catalogue afterwards, and
		// the same question giving two different sets of filters is a defect
		// nobody can reproduce.
		aiprovider.TaskQueryCompose,
		aiprovider.TaskEvaluationJudge:
		// A yes-or-no with a category. There is nothing to be imaginative
		// about, and a reclassification that changes between two identical
		// questions is a bug a person cannot reproduce.
		temperature = 0
	case aiprovider.TaskDocumentExtraction, aiprovider.TaskBillingDiagnosis:
		// Both copy values out of a record. A digit invented here reaches
		// an invoice.
		temperature = 0.1
	case aiprovider.TaskFormulaAssistant:
		temperature = 0.2
	case aiprovider.TaskAssistantChat, aiprovider.TaskGeneral:
		temperature = 0.3
	case aiprovider.TaskOperationalInsights, aiprovider.TaskDailyBriefing:
		// Prose, whose numbers are checked against the computed facts after
		// the fact, so it can afford to read like a person wrote it.
		temperature = 0.6
	}

	topP := 0.95

	return Sampling{Temperature: &temperature, TopP: &topP}
}
