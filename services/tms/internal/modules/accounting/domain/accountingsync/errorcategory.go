package accountingsync

type ErrorCategory string

const (
	ErrorCategoryTransient     = ErrorCategory("Transient")
	ErrorCategoryRateLimited   = ErrorCategory("RateLimited")
	ErrorCategoryUnauthorized  = ErrorCategory("Unauthorized")
	ErrorCategoryRevoked       = ErrorCategory("Revoked")
	ErrorCategoryConfiguration = ErrorCategory("Configuration")
	ErrorCategoryUnknown       = ErrorCategory("Unknown")
)

func (c ErrorCategory) String() string { return string(c) }

func (c ErrorCategory) IsValid() bool {
	switch c {
	case ErrorCategoryTransient,
		ErrorCategoryRateLimited,
		ErrorCategoryUnauthorized,
		ErrorCategoryRevoked,
		ErrorCategoryConfiguration,
		ErrorCategoryUnknown:
		return true
	default:
		return false
	}
}

func AllErrorCategories() []ErrorCategory {
	return []ErrorCategory{
		ErrorCategoryTransient,
		ErrorCategoryRateLimited,
		ErrorCategoryUnauthorized,
		ErrorCategoryRevoked,
		ErrorCategoryConfiguration,
		ErrorCategoryUnknown,
	}
}

const (
	JournalDayPrefix     = "jday_"
	JournalOpeningPrefix = "jopen_"
	journalDayLayout     = "20060102"
	JournalDaySQLLayout  = "YYYYMMDD"
)
