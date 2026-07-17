package report

type DefinitionKind string

const (
	DefinitionKindCustom     = DefinitionKind("custom")
	DefinitionKindCannedFork = DefinitionKind("canned_fork")
)

func (k DefinitionKind) IsValid() bool {
	return k == DefinitionKindCustom || k == DefinitionKindCannedFork
}

type Visibility string

const (
	VisibilityPrivate = Visibility("private")
	VisibilityShared  = Visibility("shared")
)

func (v Visibility) IsValid() bool {
	return v == VisibilityPrivate || v == VisibilityShared
}

type DefinitionStatus string

const (
	DefinitionStatusDraft          = DefinitionStatus("draft")
	DefinitionStatusActive         = DefinitionStatus("active")
	DefinitionStatusArchived       = DefinitionStatus("archived")
	DefinitionStatusNeedsAttention = DefinitionStatus("needs_attention")
)

func (s DefinitionStatus) IsValid() bool {
	switch s {
	case DefinitionStatusDraft, DefinitionStatusActive,
		DefinitionStatusArchived, DefinitionStatusNeedsAttention:
		return true
	default:
		return false
	}
}

type RunStatus string

const (
	RunStatusQueued    = RunStatus("queued")
	RunStatusRunning   = RunStatus("running")
	RunStatusSucceeded = RunStatus("succeeded")
	RunStatusFailed    = RunStatus("failed")
	RunStatusCanceled  = RunStatus("canceled")
	RunStatusExpired   = RunStatus("expired")
)

func (s RunStatus) IsValid() bool {
	switch s {
	case RunStatusQueued, RunStatusRunning, RunStatusSucceeded,
		RunStatusFailed, RunStatusCanceled, RunStatusExpired:
		return true
	default:
		return false
	}
}

func (s RunStatus) IsTerminal() bool {
	//nolint:exhaustive // queued/running are the non-terminal states
	switch s {
	case RunStatusSucceeded, RunStatusFailed, RunStatusCanceled, RunStatusExpired:
		return true
	default:
		return false
	}
}

type RunTrigger string

const (
	RunTriggerManual    = RunTrigger("manual")
	RunTriggerScheduled = RunTrigger("scheduled")
	RunTriggerAPI       = RunTrigger("api")
)

func (t RunTrigger) IsValid() bool {
	switch t {
	case RunTriggerManual, RunTriggerScheduled, RunTriggerAPI:
		return true
	default:
		return false
	}
}

type Format string

const (
	FormatCSV  = Format("csv")
	FormatXLSX = Format("xlsx")
	FormatPDF  = Format("pdf")
	FormatJSON = Format("json")
)

func (f Format) IsValid() bool {
	switch f {
	case FormatCSV, FormatXLSX, FormatPDF, FormatJSON:
		return true
	default:
		return false
	}
}

func (f Format) Extension() string {
	return string(f)
}

func (f Format) ContentType() string {
	switch f {
	case FormatCSV:
		return "text/csv"
	case FormatXLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case FormatPDF:
		return "application/pdf"
	case FormatJSON:
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
