package aicorrection

type Task string

const (
	TaskShipmentDraftExtraction = Task("ShipmentDraftExtraction")
	TaskInboundClassification   = Task("InboundClassification")
)

func (t Task) IsValid() bool {
	switch t {
	case TaskShipmentDraftExtraction, TaskInboundClassification:
		return true
	default:
		return false
	}
}

func (t Task) String() string { return string(t) }

func AllTasks() []Task {
	return []Task{TaskShipmentDraftExtraction, TaskInboundClassification}
}

type SourceType string

const (
	SourceDocumentShipmentDraft = SourceType("DocumentShipmentDraft")
	SourceInboundMessage        = SourceType("InboundMessage")
)

func (s SourceType) IsValid() bool {
	switch s {
	case SourceDocumentShipmentDraft, SourceInboundMessage:
		return true
	default:
		return false
	}
}

func (s SourceType) String() string { return string(s) }

func AllSourceTypes() []SourceType {
	return []SourceType{SourceDocumentShipmentDraft, SourceInboundMessage}
}

type SubjectType string

const (
	SubjectShipment       = SubjectType("Shipment")
	SubjectInboundMessage = SubjectType("InboundMessage")
)

func (s SubjectType) IsValid() bool {
	switch s {
	case SubjectShipment, SubjectInboundMessage:
		return true
	default:
		return false
	}
}

func (s SubjectType) String() string { return string(s) }

func AllSubjectTypes() []SubjectType {
	return []SubjectType{SubjectShipment, SubjectInboundMessage}
}

type Outcome string

const (
	OutcomeCorrect     = Outcome("Correct")
	OutcomeCorrected   = Outcome("Corrected")
	OutcomeMissed      = Outcome("Missed")
	OutcomeUnconfirmed = Outcome("Unconfirmed")
	OutcomeUnscored    = Outcome("Unscored")
)

func (o Outcome) IsValid() bool {
	switch o {
	case OutcomeCorrect, OutcomeCorrected, OutcomeMissed, OutcomeUnconfirmed, OutcomeUnscored:
		return true
	default:
		return false
	}
}

func (o Outcome) String() string { return string(o) }

func AllOutcomes() []Outcome {
	return []Outcome{
		OutcomeCorrect,
		OutcomeCorrected,
		OutcomeMissed,
		OutcomeUnconfirmed,
		OutcomeUnscored,
	}
}
