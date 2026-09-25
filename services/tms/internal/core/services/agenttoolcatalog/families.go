package agenttoolcatalog

import serviceports "github.com/emoss08/trenova/internal/core/ports/services"

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
	{"list_driver_settlements", "get_driver_settlement", "list_driver_pay_events"},
	{"get_ar_aging", "list_ar_open_items", "get_customer_statement"},
	{"list_edi_inbound_files", "get_edi_inbound_file", "get_edi_partner"},
	{"list_rate_agreements", "get_rate_agreement", "explain_rate"},
	{"describe_formula_schema", "test_formula_expression", "propose_formula"},
	{"accept_field", "accept_all_confident", "set_field_value"},
	{"list_locations", "set_stop_location", "set_stop_schedule", "create_location"},
	{"list_formula_templates", "set_required_field"},
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
	found []serviceports.AgentToolDescriptor,
) []serviceports.AgentToolDescriptor {
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
