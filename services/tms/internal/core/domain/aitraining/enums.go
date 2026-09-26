package aitraining

type ExportStatus string

const (
	ExportStatusQueued    = ExportStatus("Queued")
	ExportStatusRunning   = ExportStatus("Running")
	ExportStatusCompleted = ExportStatus("Completed")
	ExportStatusCanceled  = ExportStatus("Canceled")
	ExportStatusFailed    = ExportStatus("Failed")
)

func (s ExportStatus) IsValid() bool {
	switch s {
	case ExportStatusQueued,
		ExportStatusRunning,
		ExportStatusCompleted,
		ExportStatusCanceled,
		ExportStatusFailed:
		return true
	default:
		return false
	}
}

func (s ExportStatus) String() string { return string(s) }

func (s ExportStatus) IsActive() bool {
	return s == ExportStatusQueued || s == ExportStatusRunning
}

type Split string

const (
	SplitTrain      = Split("train")
	SplitValidation = Split("validation")
)

func (s Split) IsValid() bool {
	return s == SplitTrain || s == SplitValidation
}

func (s Split) String() string { return string(s) }

type DropReason string

const (
	DropConsentWithdrawn   = DropReason("consentWithdrawn")
	DropDocumentUnreadable = DropReason("documentUnreadable")
	DropNoDocumentText     = DropReason("noDocumentText")
	DropNothingConfirmed   = DropReason("nothingConfirmed")
	DropResidualIdentifier = DropReason("residualIdentifier")
)

func (r DropReason) String() string { return string(r) }
