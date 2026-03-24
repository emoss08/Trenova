package fiscalperiod

type PeriodType string

const (
	// PeriodTypeMonth is the standard monthly accounting period (periods 1–12).
	// This is the most common period type and what most carriers use.
	PeriodTypeMonth PeriodType = "Month"

	// PeriodTypeQuarter represents a quarterly accounting period. Used by
	// carriers that do quarterly accounting instead of (or in addition to)
	// monthly periods. Some smaller carriers only do quarterly hard closes.
	PeriodTypeQuarter PeriodType = "Quarter"

	// PeriodTypeWeek represents a weekly accounting period. Used by carriers
	// on a 4-4-5, 4-5-4, or 5-4-4 week-based fiscal calendar. These calendars
	// divide the year into 13 four-week periods (or 12 periods with alternating
	// 4 and 5 week lengths). Some large carriers prefer this because every
	// period has the same number of business days, making revenue comparisons
	// more meaningful than calendar-month periods where February has 28 days
	// and March has 31.
	PeriodTypeWeek PeriodType = "Week"

	// PeriodTypeAdjusting represents a special year-end adjustment period
	// (commonly called "Period 13" in Oracle, D365, and Caselle). This period
	// shares the same date range as the final operating period but is logically
	// separate. Year-end adjustments, auditor entries, and reclassifications
	// go here so they don't contaminate the operating period's numbers.
	//
	// This is critical for reporting: when running a P&L for December, the
	// controller wants to see operating results. The auditor's adjustments
	// should only appear when explicitly included. Without this separation,
	// every time an auditor posts an adjustment to Period 12, the December
	// operating numbers change — which drives controllers crazy.
	//
	// Some systems also support Period 14 for post-audit entries (Caselle),
	// but Period 13 covers 99% of use cases. If needed, a second Adjusting
	// period can be created.
	PeriodTypeAdjusting PeriodType = "Adjusting"
)

func (p PeriodType) String() string {
	return string(p)
}

func (p PeriodType) IsValid() bool {
	switch p {
	case PeriodTypeMonth, PeriodTypeQuarter, PeriodTypeWeek, PeriodTypeAdjusting:
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------
// Period Status
// ---------------------------------------------------------------
// Lifecycle: Inactive → Open → Locked → Closed → PermanentlyClosed
// Allowed reversal: Closed → Open (reopen with authorization)
// ---------------------------------------------------------------

type Status string

const (
	// PeriodStatusInactive is the initial state for periods that have been
	// created but are not yet ready for transactions. When you auto-generate
	// 12 periods for a new fiscal year, periods 2–12 start as Inactive while
	// only period 1 gets opened. Without this state, every period is immediately
	// Open the moment the year is created, which means someone could accidentally
	// post to next August.
	//
	// Oracle calls this "Never Opened". D365 uses "Not yet open".
	StatusInactive = Status("Inactive")

	// PeriodStatusOpen means the period is accepting all transactions from
	// all sources — subledger postings (AP, AR, billing pipeline), manual
	// journal entries, and system-generated entries. This is the normal
	// operating state for the current period.
	StatusOpen = Status("Open")

	// PeriodStatusLocked is the soft close state. This is the single most
	// important state that most TMS platforms get wrong or skip entirely.
	//
	// When a period is Locked:
	// - Subledger postings are BLOCKED (billing pipeline, AP vouchers,
	//   AR invoices cannot post new transactions to this period)
	// - Manual journal entries are ALLOWED (the accounting team can still
	//   post adjusting JEs, accruals, and corrections)
	//
	// This is the state a controller puts a period in when they've finished
	// the monthly close checklist. They need to stop new freight invoices
	// from landing in the period, but the accounting team still needs to
	// post accruals for delivered-but-not-yet-invoiced shipments, accrue
	// for driver settlements, and make other month-end adjustments.
	//
	// Without this state, the controller has two bad options:
	// 1. Leave the period Open while making adjustments (risk: new billing
	//    transactions keep landing, moving the target)
	// 2. Close the period, make adjustments, reopen, re-close (waste of time,
	//    clutters the audit trail)
	//
	// Oracle, D365, Successware, and NetSuite all have this state. Oracle calls
	// it "Closed" but still allows JE posting. We use "Locked" because it's
	// more intuitive — the period is locked to external sources but open
	// to the accounting team.
	StatusLocked = Status("Locked")

	// PeriodStatusClosed is the hard close state. Nothing gets in — no
	// subledger postings, no manual journal entries, nothing. The period's
	// numbers are final (for now).
	//
	// Can be reopened with proper authorization. Every major ERP supports
	// this because the real world doesn't cooperate with clean cutoffs.
	// A vendor invoice arrives 3 weeks late, an auditor finds a misclassification,
	// a customer disputes a charge that was posted to the wrong period.
	// Reopening requires a reason and is tracked for audit compliance.
	StatusClosed = Status("Closed")

	// PeriodStatusPermanentlyClosed is the terminal audit-locked state.
	// Cannot be reopened under any circumstances. This is set after the
	// external audit is complete and the period's financials are certified.
	//
	// Oracle explicitly distinguishes "Closed" from "Permanently Closed"
	// for exactly this reason. Business Central marks periods as both
	// "Closed" and "Date Locked" when permanently closed. D365 uses
	// "On Hold" for periods that cannot be reopened.
	//
	// Once permanently closed, any late transactions that should have gone
	// to this period are automatically redirected to the next open period
	// (or flagged for review, depending on the accounting control settings).
	StatusPermanentlyClosed = Status("PermanentlyClosed")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusInactive,
		StatusOpen,
		StatusLocked,
		StatusClosed,
		StatusPermanentlyClosed:
		return true
	default:
		return false
	}
}
