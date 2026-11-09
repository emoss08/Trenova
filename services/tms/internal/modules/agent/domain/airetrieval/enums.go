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

func AllIndexStatuses() []IndexStatus {
	return []IndexStatus{
		IndexStatusPending,
		IndexStatusIndexed,
		IndexStatusFailed,
		IndexStatusSkipped,
	}
}

const (
	IndexStatusPending = IndexStatus("Pending")
	IndexStatusIndexed = IndexStatus("Indexed")
	IndexStatusFailed  = IndexStatus("Failed")
	IndexStatusSkipped = IndexStatus("Skipped")
)

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

func AllUnavailableReasons() []UnavailableReason {
	return []UnavailableReason{
		UnavailableReasonExtensionMissing,
		UnavailableReasonSchemaMissing,
		UnavailableReasonTooOld,
		UnavailableReasonNoProvider,
		UnavailableReasonDisabled,
		UnavailableReasonBudgetPaused,
		UnavailableReasonNotIndexed,
		UnavailableReasonQueryTimeout,
		UnavailableReasonProviderFailed,
	}
}

const (
	UnavailableReasonExtensionMissing = UnavailableReason("ExtensionMissing")
	UnavailableReasonSchemaMissing    = UnavailableReason("SchemaMissing")
	UnavailableReasonTooOld           = UnavailableReason("TooOld")
	UnavailableReasonNoProvider       = UnavailableReason("NoProvider")
	UnavailableReasonDisabled         = UnavailableReason("Disabled")
	UnavailableReasonBudgetPaused     = UnavailableReason("BudgetPaused")
	UnavailableReasonNotIndexed       = UnavailableReason("NotIndexed")
	UnavailableReasonQueryTimeout     = UnavailableReason("QueryTimeout")
	UnavailableReasonProviderFailed   = UnavailableReason("ProviderFailed")
)

func (r UnavailableReason) IsValid() bool {
	switch r {
	case UnavailableReasonExtensionMissing,
		UnavailableReasonSchemaMissing,
		UnavailableReasonTooOld,
		UnavailableReasonNoProvider,
		UnavailableReasonDisabled,
		UnavailableReasonBudgetPaused,
		UnavailableReasonNotIndexed,
		UnavailableReasonQueryTimeout,
		UnavailableReasonProviderFailed:
		return true
	default:
		return false
	}
}

func (r UnavailableReason) String() string { return string(r) }

func (r UnavailableReason) Transient() bool {
	return r == UnavailableReasonQueryTimeout || r == UnavailableReasonProviderFailed
}

type PauseReason string

func AllPauseReasons() []PauseReason {
	return []PauseReason{PauseReasonManual, PauseReasonBudget}
}

const (
	PauseReasonManual = PauseReason("Manual")
	PauseReasonBudget = PauseReason("Budget")
)

func (r PauseReason) IsValid() bool {
	return r == PauseReasonManual || r == PauseReasonBudget
}

func (r PauseReason) String() string { return string(r) }

type CatalogCorpus string

func AllCatalogCorpora() []CatalogCorpus {
	return []CatalogCorpus{CatalogCorpusTools, CatalogCorpusProductGuide}
}

const (
	CatalogCorpusTools        = CatalogCorpus("Tools")
	CatalogCorpusProductGuide = CatalogCorpus("ProductGuide")
)

func (c CatalogCorpus) IsValid() bool {
	return c == CatalogCorpusTools || c == CatalogCorpusProductGuide
}

func (c CatalogCorpus) String() string { return string(c) }
