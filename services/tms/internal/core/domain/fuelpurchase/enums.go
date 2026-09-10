package fuelpurchase

import "github.com/shopspring/decimal"

type CardProvider string

const (
	CardProviderComdata = CardProvider("Comdata")
	CardProviderEFS     = CardProvider("EFS")
	CardProviderWEX     = CardProvider("WEX")
	CardProviderRamp    = CardProvider("Ramp")
	CardProviderOther   = CardProvider("Other")
)

func (p CardProvider) String() string { return string(p) }

func (p CardProvider) IsValid() bool {
	switch p {
	case CardProviderComdata,
		CardProviderEFS,
		CardProviderWEX,
		CardProviderRamp,
		CardProviderOther:
		return true
	default:
		return false
	}
}

func (p CardProvider) Label() string {
	switch p {
	case CardProviderComdata:
		return "Comdata"
	case CardProviderEFS:
		return "EFS"
	case CardProviderWEX:
		return "WEX"
	case CardProviderRamp:
		return "Ramp"
	case CardProviderOther:
		return "Other"
	default:
		return string(p)
	}
}

type CardStatus string

const (
	CardStatusActive    = CardStatus("Active")
	CardStatusSuspended = CardStatus("Suspended")
	CardStatusCancelled = CardStatus("Cancelled")
)

func (s CardStatus) String() string { return string(s) }

func (s CardStatus) IsValid() bool {
	switch s {
	case CardStatusActive, CardStatusSuspended, CardStatusCancelled:
		return true
	default:
		return false
	}
}

func (s CardStatus) Label() string {
	switch s {
	case CardStatusActive:
		return "Active"
	case CardStatusSuspended:
		return "Suspended"
	case CardStatusCancelled:
		return "Cancelled"
	default:
		return string(s)
	}
}

func (s CardStatus) CanTransitionTo(next CardStatus) bool {
	switch s {
	case CardStatusActive:
		return next == CardStatusSuspended || next == CardStatusCancelled
	case CardStatusSuspended:
		return next == CardStatusActive || next == CardStatusCancelled
	default:
		return false
	}
}

func (s CardStatus) IsTerminal() bool { return s == CardStatusCancelled }

type QuantityUnit string

const (
	QuantityUnitGallon = QuantityUnit("Gallon")
	QuantityUnitLitre  = QuantityUnit("Litre")
)

const gallonsScale = 3

var litresPerGallon = decimal.RequireFromString("3.785411784")

func (u QuantityUnit) String() string { return string(u) }

func (u QuantityUnit) IsValid() bool {
	return u == QuantityUnitGallon || u == QuantityUnitLitre
}

func (u QuantityUnit) Label() string {
	switch u {
	case QuantityUnitGallon:
		return "US gallons"
	case QuantityUnitLitre:
		return "Litres"
	default:
		return string(u)
	}
}

func (u QuantityUnit) ToGallons(quantity decimal.Decimal) decimal.Decimal {
	if u == QuantityUnitLitre {
		return quantity.DivRound(litresPerGallon, gallonsScale)
	}
	return quantity.Round(gallonsScale)
}

type PurchaseSource string

const (
	PurchaseSourceManual     = PurchaseSource("Manual")
	PurchaseSourceCardImport = PurchaseSource("CardImport")
)

func (s PurchaseSource) String() string { return string(s) }

func (s PurchaseSource) IsValid() bool {
	return s == PurchaseSourceManual || s == PurchaseSourceCardImport
}

func (s PurchaseSource) Label() string {
	switch s {
	case PurchaseSourceManual:
		return "Manual entry"
	case PurchaseSourceCardImport:
		return "Fuel card import"
	default:
		return string(s)
	}
}

type ImportStatus string

const (
	ImportStatusPending   = ImportStatus("Pending")
	ImportStatusParsed    = ImportStatus("Parsed")
	ImportStatusCommitted = ImportStatus("Committed")
	ImportStatusFailed    = ImportStatus("Failed")
	ImportStatusDiscarded = ImportStatus("Discarded")
)

func (s ImportStatus) String() string { return string(s) }

func (s ImportStatus) IsValid() bool {
	switch s {
	case ImportStatusPending, ImportStatusParsed, ImportStatusCommitted, ImportStatusFailed,
		ImportStatusDiscarded:
		return true
	default:
		return false
	}
}

func (s ImportStatus) Label() string {
	switch s {
	case ImportStatusPending:
		return "Awaiting file"
	case ImportStatusParsed:
		return "Ready to review"
	case ImportStatusCommitted:
		return "Imported"
	case ImportStatusFailed:
		return "Failed"
	case ImportStatusDiscarded:
		return "Discarded"
	default:
		return string(s)
	}
}

func (s ImportStatus) CanStage() bool {
	return s == ImportStatusPending || s == ImportStatusParsed || s == ImportStatusFailed
}

func (s ImportStatus) CanCommit() bool { return s == ImportStatusParsed }

func (s ImportStatus) CanDiscard() bool {
	return s == ImportStatusPending || s == ImportStatusParsed || s == ImportStatusFailed
}

func (s ImportStatus) IsTerminal() bool {
	return s == ImportStatusCommitted || s == ImportStatusDiscarded
}

type ImportRowStatus string

const (
	ImportRowStatusNew             = ImportRowStatus("New")
	ImportRowStatusDuplicateInFile = ImportRowStatus("DuplicateInFile")
	ImportRowStatusAlreadyImported = ImportRowStatus("AlreadyImported")
	ImportRowStatusError           = ImportRowStatus("Error")
	ImportRowStatusCommitted       = ImportRowStatus("Committed")
	ImportRowStatusSkipped         = ImportRowStatus("Skipped")
)

func (s ImportRowStatus) String() string { return string(s) }

func (s ImportRowStatus) IsValid() bool {
	switch s {
	case ImportRowStatusNew, ImportRowStatusDuplicateInFile, ImportRowStatusAlreadyImported,
		ImportRowStatusError, ImportRowStatusCommitted, ImportRowStatusSkipped:
		return true
	default:
		return false
	}
}

func (s ImportRowStatus) Label() string {
	switch s {
	case ImportRowStatusNew:
		return "New"
	case ImportRowStatusDuplicateInFile:
		return "Duplicate in file"
	case ImportRowStatusAlreadyImported:
		return "Already imported"
	case ImportRowStatusError:
		return "Error"
	case ImportRowStatusCommitted:
		return "Imported"
	case ImportRowStatusSkipped:
		return "Skipped"
	default:
		return string(s)
	}
}

func (s ImportRowStatus) WillCommit() bool { return s == ImportRowStatusNew }

type SourceFormat string

const (
	SourceFormatCSV        = SourceFormat("CSV")
	SourceFormatXLSX       = SourceFormat("XLSX")
	SourceFormatFixedWidth = SourceFormat("FixedWidth")
	SourceFormatAPI        = SourceFormat("API")
)

func (f SourceFormat) String() string { return string(f) }

func (f SourceFormat) IsValid() bool {
	switch f {
	case SourceFormatCSV, SourceFormatXLSX, SourceFormatFixedWidth, SourceFormatAPI:
		return true
	default:
		return false
	}
}

func (f SourceFormat) Label() string {
	switch f {
	case SourceFormatCSV:
		return "CSV"
	case SourceFormatXLSX:
		return "Excel workbook"
	case SourceFormatFixedWidth:
		return "Fixed-width file"
	case SourceFormatAPI:
		return "Provider API"
	default:
		return string(f)
	}
}

// ImportOrigin separates a batch a person opened by choosing a file from one a
// scheduled sync opened on its own. A feed batch has no uploader and no stored
// document, so the lifecycle rules that require those only apply to uploads.
type ImportOrigin string

const (
	ImportOriginUpload = ImportOrigin("Upload")
	ImportOriginFeed   = ImportOrigin("Feed")
)

func (o ImportOrigin) String() string { return string(o) }

func (o ImportOrigin) IsValid() bool {
	return o == ImportOriginUpload || o == ImportOriginFeed
}

func (o ImportOrigin) IsFeed() bool { return o == ImportOriginFeed }

func (o ImportOrigin) Label() string {
	switch o {
	case ImportOriginUpload:
		return "Uploaded"
	case ImportOriginFeed:
		return "Synced"
	default:
		return string(o)
	}
}
