package agentexception

type Severity string

const (
	SeverityLow      = Severity("Low")
	SeverityMedium   = Severity("Medium")
	SeverityHigh     = Severity("High")
	SeverityCritical = Severity("Critical")
)

func (s Severity) IsValid() bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	default:
		return false
	}
}

func SeverityValues() []Severity {
	return []Severity{SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical}
}

type ResolutionState string

const (
	ResolutionStateOpen      = ResolutionState("Open")
	ResolutionStateInReview  = ResolutionState("InReview")
	ResolutionStateResolved  = ResolutionState("Resolved")
	ResolutionStateDismissed = ResolutionState("Dismissed")
)

func (s ResolutionState) IsValid() bool {
	switch s {
	case ResolutionStateOpen,
		ResolutionStateInReview,
		ResolutionStateResolved,
		ResolutionStateDismissed:
		return true
	default:
		return false
	}
}

type ExceptionCategory string

const (
	CategoryMissingDocumentation       = ExceptionCategory("MissingDocumentation")
	CategoryIncorrectRates             = ExceptionCategory("IncorrectRates")
	CategoryWeightDiscrepancy          = ExceptionCategory("WeightDiscrepancy")
	CategoryAccessorialDispute         = ExceptionCategory("AccessorialDispute")
	CategoryDuplicateCharge            = ExceptionCategory("DuplicateCharge")
	CategoryMissingReferenceNumber     = ExceptionCategory("MissingReferenceNumber")
	CategoryCustomerInformationError   = ExceptionCategory("CustomerInformationError")
	CategoryServiceFailure             = ExceptionCategory("ServiceFailure")
	CategoryRateNotOnFile              = ExceptionCategory("RateNotOnFile")
	CategoryMissingBOL                 = ExceptionCategory("MissingBOL")
	CategoryRateMissingBasis           = ExceptionCategory("RateMissingBasis")
	CategoryRateVarianceRequiresAction = ExceptionCategory("RateVarianceRequiresAction")
	CategoryUnresolvedServiceFailures  = ExceptionCategory("UnresolvedServiceFailures")
	CategoryMissingRequiredDocument    = ExceptionCategory("MissingRequiredDocument")
	CategoryConfidenceBelowThreshold   = ExceptionCategory("ConfidenceBelowThreshold")
	CategoryUnableToDiagnose           = ExceptionCategory("UnableToDiagnose")
	CategoryOther                      = ExceptionCategory("Other")
)

func AllExceptionCategories() []ExceptionCategory {
	return []ExceptionCategory{
		CategoryMissingDocumentation,
		CategoryIncorrectRates,
		CategoryWeightDiscrepancy,
		CategoryAccessorialDispute,
		CategoryDuplicateCharge,
		CategoryMissingReferenceNumber,
		CategoryCustomerInformationError,
		CategoryServiceFailure,
		CategoryRateNotOnFile,
		CategoryMissingBOL,
		CategoryRateMissingBasis,
		CategoryRateVarianceRequiresAction,
		CategoryUnresolvedServiceFailures,
		CategoryMissingRequiredDocument,
		CategoryConfidenceBelowThreshold,
		CategoryUnableToDiagnose,
		CategoryOther,
	}
}

func (c ExceptionCategory) IsValid() bool {
	switch c {
	case CategoryMissingDocumentation,
		CategoryIncorrectRates,
		CategoryWeightDiscrepancy,
		CategoryAccessorialDispute,
		CategoryDuplicateCharge,
		CategoryMissingReferenceNumber,
		CategoryCustomerInformationError,
		CategoryServiceFailure,
		CategoryRateNotOnFile,
		CategoryMissingBOL,
		CategoryRateMissingBasis,
		CategoryRateVarianceRequiresAction,
		CategoryUnresolvedServiceFailures,
		CategoryMissingRequiredDocument,
		CategoryConfidenceBelowThreshold,
		CategoryUnableToDiagnose,
		CategoryOther:
		return true
	default:
		return false
	}
}

func ExceptionCategoryFromValidationCode(code string) ExceptionCategory {
	switch code {
	case "missing_bol":
		return CategoryMissingBOL
	case "rate_missing_basis":
		return CategoryRateMissingBasis
	case "rate_variance_requires_action":
		return CategoryRateVarianceRequiresAction
	case "unresolved_service_failures":
		return CategoryUnresolvedServiceFailures
	case "missing_required_document":
		return CategoryMissingRequiredDocument
	default:
		return CategoryOther
	}
}
