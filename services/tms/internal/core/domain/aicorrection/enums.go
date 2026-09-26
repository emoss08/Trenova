package aicorrection

type Task string

const (
	TaskShipmentDraftExtraction = Task("ShipmentDraftExtraction")
)

func (t Task) IsValid() bool {
	switch t {
	case TaskShipmentDraftExtraction:
		return true
	default:
		return false
	}
}

func (t Task) String() string { return string(t) }

func AllTasks() []Task {
	return []Task{TaskShipmentDraftExtraction}
}

type SourceType string

const (
	SourceDocumentShipmentDraft = SourceType("DocumentShipmentDraft")
)

func (s SourceType) IsValid() bool {
	switch s {
	case SourceDocumentShipmentDraft:
		return true
	default:
		return false
	}
}

func (s SourceType) String() string { return string(s) }

func AllSourceTypes() []SourceType {
	return []SourceType{SourceDocumentShipmentDraft}
}

type SubjectType string

const (
	SubjectShipment = SubjectType("Shipment")
)

func (s SubjectType) IsValid() bool {
	switch s {
	case SubjectShipment:
		return true
	default:
		return false
	}
}

func (s SubjectType) String() string { return string(s) }

func AllSubjectTypes() []SubjectType {
	return []SubjectType{SubjectShipment}
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
