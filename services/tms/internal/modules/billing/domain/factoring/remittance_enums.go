package factoring

type RemittanceStatus string

const (
	RemittanceStatusPending          = RemittanceStatus("Pending")
	RemittanceStatusNeedsMapping     = RemittanceStatus("NeedsMapping")
	RemittanceStatusParsed           = RemittanceStatus("Parsed")
	RemittanceStatusFailed           = RemittanceStatus("Failed")
	RemittanceStatusPartiallyApplied = RemittanceStatus("PartiallyApplied")
	RemittanceStatusApplied          = RemittanceStatus("Applied")
	RemittanceStatusDiscarded        = RemittanceStatus("Discarded")
)

func AllRemittanceStatuses() []RemittanceStatus {
	return []RemittanceStatus{
		RemittanceStatusPending,
		RemittanceStatusNeedsMapping,
		RemittanceStatusParsed,
		RemittanceStatusFailed,
		RemittanceStatusPartiallyApplied,
		RemittanceStatusApplied,
		RemittanceStatusDiscarded,
	}
}

func (s RemittanceStatus) IsValid() bool {
	switch s {
	case RemittanceStatusPending,
		RemittanceStatusNeedsMapping,
		RemittanceStatusParsed,
		RemittanceStatusFailed,
		RemittanceStatusPartiallyApplied,
		RemittanceStatusApplied,
		RemittanceStatusDiscarded:
		return true
	}
	return false
}

func (s RemittanceStatus) String() string { return string(s) }

func (s RemittanceStatus) CanStage() bool {
	switch s {
	case RemittanceStatusPending,
		RemittanceStatusNeedsMapping,
		RemittanceStatusParsed,
		RemittanceStatusFailed:
		return true
	case RemittanceStatusPartiallyApplied, RemittanceStatusApplied, RemittanceStatusDiscarded:
	}
	return false
}

func (s RemittanceStatus) CanEditLines() bool {
	return s == RemittanceStatusParsed || s == RemittanceStatusPartiallyApplied
}

func (s RemittanceStatus) CanApply() bool { return s.CanEditLines() }

func (s RemittanceStatus) CanDiscard() bool { return s.CanStage() }

func (s RemittanceStatus) IsOpen() bool {
	switch s {
	case RemittanceStatusNeedsMapping, RemittanceStatusParsed, RemittanceStatusPartiallyApplied:
		return true
	case RemittanceStatusPending,
		RemittanceStatusFailed,
		RemittanceStatusApplied,
		RemittanceStatusDiscarded:
	}
	return false
}

type RemittanceSource string

const (
	RemittanceSourceUpload = RemittanceSource("Upload")
	RemittanceSourceEmail  = RemittanceSource("Email")
)

func (s RemittanceSource) IsValid() bool {
	return s == RemittanceSourceUpload || s == RemittanceSourceEmail
}

func (s RemittanceSource) String() string { return string(s) }

type RemittanceFormat string

const (
	RemittanceFormatCsv  = RemittanceFormat("Csv")
	RemittanceFormatXlsx = RemittanceFormat("Xlsx")
	RemittanceFormatPdf  = RemittanceFormat("Pdf")
)

func (f RemittanceFormat) IsValid() bool {
	switch f {
	case RemittanceFormatCsv, RemittanceFormatXlsx, RemittanceFormatPdf:
		return true
	}
	return false
}

func (f RemittanceFormat) String() string { return string(f) }

type RemittanceLineKind string

const (
	RemittanceLineFunding        = RemittanceLineKind("Funding")
	RemittanceLineRejection      = RemittanceLineKind("Rejection")
	RemittanceLineReserveRelease = RemittanceLineKind("ReserveRelease")
	RemittanceLineChargeback     = RemittanceLineKind("Chargeback")
	RemittanceLineOther          = RemittanceLineKind("Other")
)

func AllRemittanceLineKinds() []RemittanceLineKind {
	return []RemittanceLineKind{
		RemittanceLineFunding,
		RemittanceLineRejection,
		RemittanceLineReserveRelease,
		RemittanceLineChargeback,
		RemittanceLineOther,
	}
}

func (k RemittanceLineKind) IsValid() bool {
	switch k {
	case RemittanceLineFunding,
		RemittanceLineRejection,
		RemittanceLineReserveRelease,
		RemittanceLineChargeback,
		RemittanceLineOther:
		return true
	}
	return false
}

func (k RemittanceLineKind) String() string { return string(k) }

func (k RemittanceLineKind) Books() bool {
	return k.IsValid() && k != RemittanceLineOther
}

type RemittanceMatchStatus string

const (
	RemittanceLineMatched   = RemittanceMatchStatus("Matched")
	RemittanceLineSuggested = RemittanceMatchStatus("Suggested")
	RemittanceLineUnmatched = RemittanceMatchStatus("Unmatched")
	RemittanceLineIgnored   = RemittanceMatchStatus("Ignored")
	RemittanceLineError     = RemittanceMatchStatus("Error")
	RemittanceLineApplied   = RemittanceMatchStatus("Applied")
)

func AllRemittanceMatchStatuses() []RemittanceMatchStatus {
	return []RemittanceMatchStatus{
		RemittanceLineMatched,
		RemittanceLineSuggested,
		RemittanceLineUnmatched,
		RemittanceLineIgnored,
		RemittanceLineError,
		RemittanceLineApplied,
	}
}

func (s RemittanceMatchStatus) IsValid() bool {
	switch s {
	case RemittanceLineMatched,
		RemittanceLineSuggested,
		RemittanceLineUnmatched,
		RemittanceLineIgnored,
		RemittanceLineError,
		RemittanceLineApplied:
		return true
	}
	return false
}

func (s RemittanceMatchStatus) String() string { return string(s) }

func (s RemittanceMatchStatus) IsException() bool {
	return s == RemittanceLineSuggested || s == RemittanceLineUnmatched ||
		s == RemittanceLineError
}

func (s RemittanceMatchStatus) IsSettled() bool {
	return s == RemittanceLineApplied || s == RemittanceLineIgnored
}

type RemittanceMatchedBy string

const (
	RemittanceMatchedByAuto   = RemittanceMatchedBy("Auto")
	RemittanceMatchedByPerson = RemittanceMatchedBy("Person")
)

func (m RemittanceMatchedBy) IsValid() bool {
	return m == RemittanceMatchedByAuto || m == RemittanceMatchedByPerson
}

func (m RemittanceMatchedBy) String() string { return string(m) }

type RemittanceMatchReason string

const (
	MatchReasonInvoiceAmountDate       = RemittanceMatchReason("InvoiceAmountDate")
	MatchReasonInvoiceAmount           = RemittanceMatchReason("InvoiceAmount")
	MatchReasonInvoiceNearAmount       = RemittanceMatchReason("InvoiceNearAmount")
	MatchReasonPartialInvoiceAmount    = RemittanceMatchReason("PartialInvoiceAmount")
	MatchReasonInvoiceOnly             = RemittanceMatchReason("InvoiceOnly")
	MatchReasonPartialInvoiceNear      = RemittanceMatchReason("PartialInvoiceNearAmount")
	MatchReasonAmountOnly              = RemittanceMatchReason("AmountOnly")
	MatchReasonNearAmountOnly          = RemittanceMatchReason("NearAmountOnly")
	MatchReasonClaimedByAnotherLine    = RemittanceMatchReason("ClaimedByAnotherLine")
	MatchReasonChosenByPerson          = RemittanceMatchReason("ChosenByPerson")
	MatchReasonNoOpenItem              = RemittanceMatchReason("NoOpenItem")
	MatchReasonNotBookedFromRemittance = RemittanceMatchReason("NotBookedFromRemittance")
)

func AllRemittanceMatchReasons() []RemittanceMatchReason {
	return []RemittanceMatchReason{
		MatchReasonInvoiceAmountDate,
		MatchReasonInvoiceAmount,
		MatchReasonInvoiceNearAmount,
		MatchReasonPartialInvoiceAmount,
		MatchReasonInvoiceOnly,
		MatchReasonPartialInvoiceNear,
		MatchReasonAmountOnly,
		MatchReasonNearAmountOnly,
		MatchReasonClaimedByAnotherLine,
		MatchReasonChosenByPerson,
		MatchReasonNoOpenItem,
		MatchReasonNotBookedFromRemittance,
	}
}

func (r RemittanceMatchReason) IsValid() bool {
	for _, known := range AllRemittanceMatchReasons() {
		if known == r {
			return true
		}
	}
	return false
}

func (r RemittanceMatchReason) String() string { return string(r) }
