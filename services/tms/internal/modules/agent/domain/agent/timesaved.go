package agent

// Minutes a person would have spent doing by hand what one approved run of a
// tool did.
//
// These are estimates and the scorecard says so. They exist because the
// alternative — reporting only "142 proposals approved" — asks a reader to do
// the arithmetic that decides whether an agent is worth keeping, and most
// people will not. A figure with a stated basis and a visible label beats a
// figure nobody computes.
//
// The basis is the clerical work the tool replaces, not the thinking: finding
// the record, opening it, typing the change, saving it, and noting it
// somewhere. It deliberately excludes deciding whether the change was right,
// because a person still does that — they approved the proposal.
//
// A tool with no entry is worth the default rather than nothing. Zero would
// quietly claim that a write nobody bothered to price saves no time, which is
// a stronger claim than the silence deserves.
const (
	// DefaultMinutesSaved covers an unlisted write: find the record, change
	// one field, save.
	DefaultMinutesSaved = 3

	// A read tool saves nothing on its own. Looking something up is what the
	// person asked for, not work the agent took off them, and counting it
	// would let a chatty read-only agent out-score one that does the work.
	readMinutesSaved = 0
)

// toolMinutesSaved prices the tools worth more or less than the default.
var toolMinutesSaved = map[string]int{
	// Dispatch. Covering a move means reading the board, checking hours and
	// equipment, and telling somebody.
	"assign_move":           12,
	"plan_dispatch":         20,
	"rank_move_candidates":  8,
	"place_shipment_hold":   4,
	"release_shipment_hold": 4,
	"record_stop_actual":    2,
	"cancel_shipment":       6,
	"create_shipment":       15,
	"quote_shipment":        8,

	// Billing. A charge correction is a hunt through the rate and the
	// invoice before anything is typed.
	"correct_charge_code":            10,
	"post_customer_payment":          6,
	"match_bank_receipt":             8,
	"resolve_bank_receipt_work_item": 6,
	"attach_document_to_shipment":    3,
	"request_missing_docs":           5,

	// Communication. Writing the message is most of the cost.
	"email_customer":       8,
	"notify_driver":        4,
	"add_shipment_comment": 2,

	// Compliance and safety.
	"approve_worker_pto":        3,
	"reject_worker_pto":         3,
	"cancel_worker_pto":         3,
	"resolve_service_failure":   10,
	"evaluate_service_failures": 15,

	// Reporting. Building a definition by hand is the expensive one; running
	// a saved report is a click.
	"create_report": 25,
	"fork_report":   10,
	"run_report":    1,

	// Bookkeeping the agent does on itself. Real, but small.
	"dismiss_insight":        1,
	"raise_exception":        readMinutesSaved,
	"flag_for_manual_review": readMinutesSaved,
	"remember":               readMinutesSaved,
	"forget_memory":          readMinutesSaved,
}

// MinutesSavedFor is what one carried-out run of a tool is worth.
func MinutesSavedFor(toolName string) int {
	if minutes, ok := toolMinutesSaved[toolName]; ok {
		return minutes
	}

	return DefaultMinutesSaved
}
