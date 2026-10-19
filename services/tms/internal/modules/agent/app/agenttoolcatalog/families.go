package agenttoolcatalog

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

// maxFamilySize bounds a family. A find_tools answer is kept short on
// purpose, and a family that drags in ten tools re-creates the list it was
// narrowed from.
const maxFamilySize = 4

// families are tools that are used together, so finding one makes the rest
// callable too.
//
// A search matches words, and a model that asked for "run the report" was
// handed run_report alone. Its first move with it was to look the report up,
// which took a second find_tools for list_reports and a third for
// describe_report: three round trips for one step of work. A family is not a
// grant — a member the agent does not hold is still never offered — it only
// saves the model from searching for something it will need next.
var families = [...][]string{
	{"list_reports", "describe_report", "run_report", "get_report_run"},
	{"list_report_datasets", "describe_report_dataset", "preview_report"},
	{
		"get_accounting_sync_status",
		"check_accounting_connection",
		"pause_accounting_sync",
		"resume_accounting_sync",
	},
	{
		"list_accounting_sync_records",
		"get_accounting_sync_record",
		"retry_accounting_sync",
		"skip_accounting_sync",
	},
	{
		"list_accounting_mapping_gaps",
		"get_accounting_mapping",
		"set_accounting_mapping",
		"create_accounting_reference_record",
	},
	{"list_watchtower_items", "get_daily_briefing", "list_agent_runs", "get_agent_run"},
	{"settlement_list_driver", "settlement_get_driver", "settlement_list_pay_events"},
	{"receivable_get_aging", "receivable_list_open_items", "receivable_get_statement"},
	{
		"list_factoring_ready_invoices",
		"get_factoring_readiness",
		"request_missing_factoring_document",
		"set_invoice_factoring_override",
	},
	{
		"list_factoring_noas",
		"draft_factoring_noa",
		"send_factoring_noa",
		"record_factoring_noa_acknowledgement",
	},
	{
		"list_factoring_submissions",
		"submit_invoices_to_factor",
		"retry_factoring_submission",
		"cancel_factoring_submission",
	},
	{"get_factoring_submission", "record_factoring_funding"},
	{
		"get_factoring_reserve_balance",
		"release_factoring_reserve",
		"record_factoring_chargeback",
		"reverse_factoring_chargeback",
	},
	{"list_factoring_remittances", "get_factoring_remittance", "discard_factoring_remittance"},
	{
		"resolve_factoring_remittance_line",
		"rematch_factoring_remittance",
		"apply_factoring_remittance",
	},
	{"list_edi_inbound_files", "get_edi_inbound_file", "get_edi_partner"},
	{"list_edi_transfers", "get_edi_transfer", "accept_edi_tender", "decline_edi_tender"},
	{"list_edi_tender_changes", "review_edi_tender_change"},
	{"list_edi_transfer_changes", "review_edi_transfer_change"},
	{
		"list_edi_messages",
		"get_edi_message",
		"retry_edi_message_delivery",
		"replay_edi_message",
	},
	{"preview_edi_document", "send_edi_status_update"},
	{"draft_edi_connection", "apply_edi_connection_draft"},
	{"list_rate_agreements", "get_rate_agreement", "explain_rate"},
	{"describe_formula_schema", "test_formula_expression", "propose_formula"},
	{"intake_accept_field", "intake_accept_confident", "intake_set_field"},
	{"list_locations", "intake_set_stop_location", "intake_set_stop_schedule", "create_location"},
	{"list_formula_templates", "get_formula_template", "intake_set_required_field"},
	{
		"billing_list_queue_items",
		"billing_get_queue_item",
		"billing_assign_biller",
		"billing_start_review",
	},
	{"billing_approve_queue_item", "invoice_post", "invoice_send", "invoice_get"},
	{"quote_shipment", "shipment_create", "shipment_duplicate"},
	{"get_dispatch_board", "rank_move_candidates", "assign_move", "assign_move_to_carrier"},
	{"invoice_list_runs", "invoice_get_run", "invoice_commit_run", "invoice_cancel_run"},
	{
		"list_carrier_intel_events",
		"get_carrier_intel_event",
		"acknowledge_carrier_intel_event",
		"resolve_carrier_intel_event",
	},
	{
		"draft_inbound_reply",
		"discard_inbound_reply_draft",
		"reply_to_inbound_message",
		"wait_for_inbound_reply",
	},
	{"list_triage_assignees", "assign_inbound_messages", "hand_off_triage"},
	{"snooze_inbound_messages", "wake_inbound_message", "reopen_inbound_messages"},
	{"draft_inbound_load", "create_inbound_load", "discard_inbound_load"},
	{
		"file_inbound_attachment",
		"unfile_inbound_attachment",
		"resolve_inbound_conflict",
		"reopen_inbound_conflict",
	},
	{"block_inbound_sender", "unblock_inbound_sender"},
	{"link_inbound_message", "unlink_inbound_message"},
}

// indexFamilies maps each family member to the others, in family order.
func indexFamilies() map[string][]string {
	index := make(map[string][]string, len(families)*maxFamilySize)
	for _, family := range families {
		for _, name := range family {
			others := make([]string, 0, len(family)-1)
			for _, other := range family {
				if other != name {
					others = append(others, other)
				}
			}
			index[name] = append(index[name], others...)
		}
	}

	return index
}

// withFamilies appends, after the tools a search found, the members of their
// families the search did not reach. A member outside allowed, or missing
// from the catalog, is skipped: the family widens the answer, never the grant.
func (c *Catalog) withFamilies(
	allowed map[string]struct{},
	found []agentports.AgentToolDescriptor,
) []agentports.AgentToolDescriptor {
	if len(found) == 0 {
		return found
	}

	present := make(map[string]struct{}, len(found)+maxFamilySize)
	for idx := range found {
		present[found[idx].Name] = struct{}{}
	}

	matched := len(found)
	for idx := range matched {
		for _, member := range c.family[found[idx].Name] {
			if _, ok := present[member]; ok {
				continue
			}
			if allowed != nil {
				if _, ok := allowed[member]; !ok {
					continue
				}
			}
			position, ok := c.byName[member]
			if !ok {
				continue
			}
			present[member] = struct{}{}
			found = append(found, c.entries[position].descriptor)
		}
	}

	return found
}
