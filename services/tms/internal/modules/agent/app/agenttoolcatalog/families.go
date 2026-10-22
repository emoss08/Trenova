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
// handed report_run alone. Its first move with it was to look the report up,
// which took a second find_tools for report_list and a third for
// report_get: three round trips for one step of work. A family is not a
// grant — a member the agent does not hold is still never offered — it only
// saves the model from searching for something it will need next.
var families = [...][]string{
	{"report_list", "report_get", "report_run", "report_get_run"},
	{"report_list_datasets", "report_get_dataset", "report_preview"},
	{
		"accounting_get_connection",
		"accounting_check_connection",
		"accounting_pause_sync",
		"accounting_resume_sync",
	},
	{
		"accounting_list_sync_records",
		"accounting_get_sync_record",
		"accounting_retry_sync_record",
		"accounting_skip_sync_record",
	},
	{
		"accounting_list_mapping_gaps",
		"accounting_get_mapping",
		"accounting_set_mapping",
		"accounting_create_reference_record",
	},
	{
		"exception_list_watchtower_items",
		"insight_get_daily_briefing",
		"agent_list_runs",
		"agent_get_run",
	},
	{"settlement_list_driver", "settlement_get_driver", "settlement_list_pay_events"},
	{"receivable_get_aging", "receivable_list_open_items", "receivable_get_statement"},
	{
		"factoring_list_readiness",
		"factoring_get_readiness",
		"factoring_request_document",
		"factoring_set_override",
	},
	{
		"factoring_list_noas",
		"factoring_create_noa",
		"factoring_send_noa",
		"factoring_record_noa_acknowledgement",
	},
	{
		"factoring_list_submissions",
		"factoring_create_submission",
		"factoring_retry_submission",
		"factoring_cancel_submission",
	},
	{"factoring_get_submission", "factoring_record_funding"},
	{
		"factoring_get_reserve_balance",
		"factoring_release_reserve",
		"factoring_record_chargeback",
		"factoring_reverse_chargeback",
	},
	{"factoring_list_remittances", "factoring_get_remittance", "factoring_discard_remittance"},
	{
		"factoring_resolve_remittance_line",
		"factoring_rematch_remittance",
		"factoring_apply_remittance",
	},
	{"edi_list_inbound_files", "edi_get_inbound_file", "edi_get_partner"},
	{"edi_list_transfers", "edi_get_transfer", "edi_accept_tender", "edi_decline_tender"},
	{"edi_list_tender_changes", "edi_review_tender_change"},
	{"edi_list_transfer_changes", "edi_review_transfer_change"},
	{
		"edi_list_messages",
		"edi_get_message",
		"edi_retry_message",
		"edi_replay_message",
	},
	{"edi_preview_x12", "edi_send_status_update"},
	{"edi_draft_connection", "edi_apply_connection_draft"},
	{"rate_list_agreements", "rate_get_agreement", "rate_explain"},
	{"formula_describe_schema", "formula_test_expression", "formula_propose"},
	{
		"intake_accept_field",
		"intake_accept_confident",
		"intake_set_field",
		"intake_set_required_field",
	},
	{"location_list", "intake_set_stop_location", "intake_set_stop_schedule", "location_create"},
	{"formula_list_templates", "formula_get_template"},
	{
		"billing_list_queue_items",
		"billing_get_queue_item",
		"billing_assign_biller",
		"billing_start_review",
	},
	{"billing_approve_queue_item", "invoice_post", "invoice_send", "invoice_get"},
	{"rate_quote", "shipment_create", "shipment_duplicate"},
	{
		"dispatch_get_board",
		"dispatch_rank_candidates",
		"dispatch_assign_move",
		"dispatch_create_carrier_assignment",
	},
	{"invoice_list_runs", "invoice_get_run", "invoice_commit_run", "invoice_cancel_run"},
	{
		"carrier_list_intelligence_events",
		"carrier_get_intelligence_event",
		"carrier_acknowledge_intelligence_event",
		"carrier_resolve_intelligence_event",
	},
	{
		"inbox_draft_reply",
		"inbox_discard_reply_draft",
		"inbox_send_reply",
		"inbox_await_reply",
	},
	{"inbox_list_triage_assignees", "inbox_assign_inbound_message", "inbox_assign_queue"},
	{"inbox_snooze_inbound_message", "inbox_wake_inbound_message", "inbox_reopen_inbound_message"},
	{"inbox_draft_load", "inbox_create_load", "inbox_discard_load"},
	{
		"inbox_file_attachment",
		"inbox_unfile_attachment",
		"inbox_resolve_conflict",
		"inbox_reopen_conflict",
	},
	{"inbox_block_sender", "inbox_unblock_sender"},
	{"inbox_link_inbound_message", "inbox_unlink_inbound_message"},
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
