package airetrieval

type SourceType string

const (
	SourceTypeMemory         = SourceType("Memory")
	SourceTypeDocument       = SourceType("Document")
	SourceTypeInboundMessage = SourceType("InboundMessage")
)

func AllSourceTypes() []SourceType {
	return []SourceType{SourceTypeMemory, SourceTypeDocument, SourceTypeInboundMessage}
}

func (s SourceType) IsValid() bool {
	switch s {
	case SourceTypeMemory, SourceTypeDocument, SourceTypeInboundMessage:
		return true
	default:
		return false
	}
}

func (s SourceType) String() string { return string(s) }

type IndexStatus string

const (
	IndexStatusPending = IndexStatus("Pending")
	IndexStatusIndexed = IndexStatus("Indexed")
	IndexStatusFailed  = IndexStatus("Failed")
	IndexStatusSkipped = IndexStatus("Skipped")
)

func AllIndexStatuses() []IndexStatus {
	return []IndexStatus{
		IndexStatusPending,
		IndexStatusIndexed,
		IndexStatusFailed,
		IndexStatusSkipped,
	}
}

func (s IndexStatus) IsValid() bool {
	switch s {
	case IndexStatusPending, IndexStatusIndexed, IndexStatusFailed, IndexStatusSkipped:
		return true
	default:
		return false
	}
}

func (s IndexStatus) IsTerminal() bool {
	return s == IndexStatusIndexed || s == IndexStatusFailed || s == IndexStatusSkipped
}

func (s IndexStatus) String() string { return string(s) }

type UnavailableReason string

const (
	UnavailableReasonExtensionMissing = UnavailableReason("ExtensionMissing")
	UnavailableReasonSchemaMissing    = UnavailableReason("SchemaMissing")
	UnavailableReasonTooOld           = UnavailableReason("TooOld")
	UnavailableReasonNoProvider       = UnavailableReason("NoProvider")
	UnavailableReasonDisabled         = UnavailableReason("Disabled")
	UnavailableReasonBudgetPaused     = UnavailableReason("BudgetPaused")
)

func AllUnavailableReasons() []UnavailableReason {
	return []UnavailableReason{
		UnavailableReasonExtensionMissing,
		UnavailableReasonSchemaMissing,
		UnavailableReasonTooOld,
		UnavailableReasonNoProvider,
		UnavailableReasonDisabled,
		UnavailableReasonBudgetPaused,
	}
}

func (r UnavailableReason) IsValid() bool {
	switch r {
	case UnavailableReasonExtensionMissing,
		UnavailableReasonSchemaMissing,
		UnavailableReasonTooOld,
		UnavailableReasonNoProvider,
		UnavailableReasonDisabled,
		UnavailableReasonBudgetPaused:
		return true
	default:
		return false
	}
}

func (r UnavailableReason) String() string { return string(r) }

type PauseReason string

const (
	PauseReasonManual = PauseReason("Manual")
	PauseReasonBudget = PauseReason("Budget")
)

func AllPauseReasons() []PauseReason {
	return []PauseReason{PauseReasonManual, PauseReasonBudget}
}

func (r PauseReason) IsValid() bool {
	return r == PauseReasonManual || r == PauseReasonBudget
}

func (r PauseReason) String() string { return string(r) }

type CatalogCorpus string

const (
	CatalogCorpusTools        = CatalogCorpus("Tools")
	CatalogCorpusProductGuide = CatalogCorpus("ProductGuide")
)

func AllCatalogCorpora() []CatalogCorpus {
	return []CatalogCorpus{CatalogCorpusTools, CatalogCorpusProductGuide}
}

func (c CatalogCorpus) IsValid() bool {
	return c == CatalogCorpusTools || c == CatalogCorpusProductGuide
}

func (c CatalogCorpus) String() string { return string(c) }
