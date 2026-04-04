package documentparsingrule

type DocumentKind string

const (
	DocumentKindRateConfirmation DocumentKind = "RateConfirmation"
)

type VersionStatus string

const (
	VersionStatusDraft     VersionStatus = "Draft"
	VersionStatusPublished VersionStatus = "Published"
	VersionStatusArchived  VersionStatus = "Archived"
)

type ParserMode string

const (
	ParserModeMergeWithBase ParserMode = "merge_with_base"
	ParserModeOverrideBase  ParserMode = "override_base"
)
