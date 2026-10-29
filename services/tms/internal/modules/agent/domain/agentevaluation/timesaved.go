package agentevaluation

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
	"dispatch_assign_move":        12,
	"dispatch_plan":               20,
	"dispatch_rank_candidates":    8,
	"shipment_place_hold":         4,
	"shipment_release_hold":       4,
	"dispatch_record_stop_actual": 2,
	"dispatch_record_check_call":  3,
	"shipment_cancel":             6,
	"shipment_create":             15,
	"rate_quote":                  8,

	// Billing. A charge correction is a hunt through the rate and the
	// invoice before anything is typed.
	"billing_correct_charge_codes":              10,
	"receivable_post_customer_payment":          6,
	"receivable_match_bank_receipt":             8,
	"receivable_resolve_bank_receipt_work_item": 6,
	"shipment_attach_document":                  3,
	"shipment_request_documents":                5,

	// Communication. Writing the message is most of the cost.
	"customer_send_email":  8,
	"worker_send_message":  4,
	"shipment_add_comment": 2,

	// Compliance and safety.
	"worker_approve_pto":                  3,
	"worker_reject_pto":                   3,
	"worker_cancel_pto":                   3,
	"exception_resolve_service_failure":   10,
	"exception_evaluate_service_failures": 15,

	// Reporting. Building a definition by hand is the expensive one; running
	// a saved report is a click.
	"report_create": 25,
	"report_fork":   10,
	"report_run":    1,

	// Bookkeeping the agent does on itself. Real, but small.
	"insight_dismiss":       1,
	"agent_raise_exception": readMinutesSaved,
	"agent_create_memory":   readMinutesSaved,
	"agent_retire_memory":   readMinutesSaved,
}

// MinutesSavedFor is what one carried-out run of a tool is worth.
func MinutesSavedFor(toolName string) int {
	if minutes, ok := toolMinutesSaved[toolName]; ok {
		return minutes
	}

	return DefaultMinutesSaved
}
