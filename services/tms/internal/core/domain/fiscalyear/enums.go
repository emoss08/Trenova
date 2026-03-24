package fiscalyear

// ---------------------------------------------------------------
// Fiscal Year Status
// ---------------------------------------------------------------
// Lifecycle: Draft → Open → Closed → PermanentlyClosed
// Allowed reversal: Closed → Open (reopen with authorization)
// ---------------------------------------------------------------

type Status string

const (
	// FiscalYearStatusDraft is the initial state when a fiscal year is created.
	// The year and its periods are being configured. No transactions can be
	// posted to any period in this year. The accounting team uses this phase
	// to set up period dates, validate the calendar, and configure controls
	// before going live.
	StatusDraft = Status("Draft")

	// FiscalYearStatusOpen means the fiscal year is active and accepting
	// transactions. At least one period within the year must be in Open or
	// Locked status. This is the normal operating state for the current
	// fiscal year.
	StatusOpen = Status("Open")

	// FiscalYearStatusClosed means year-end close has been completed.
	// Retained earnings have been calculated and income statement accounts
	// have been zeroed. Can be reopened with proper authorization for
	// material corrections — this is standard practice in Oracle GL, D365,
	// NetSuite, and Business Central. Reopening requires a reason and is
	// tracked for audit compliance.
	StatusClosed = Status("Closed")

	// FiscalYearStatusPermanentlyClosed is the terminal audit-locked state.
	// Cannot be reopened under any circumstances. This is set after external
	// auditors have signed off and the year's financials are final. Oracle GL
	// explicitly distinguishes this from regular Closed for exactly this reason.
	StatusPermanentlyClosed = Status("PermanentlyClosed")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft,
		StatusOpen,
		StatusClosed,
		StatusPermanentlyClosed:
		return true
	default:
		return false
	}
}
