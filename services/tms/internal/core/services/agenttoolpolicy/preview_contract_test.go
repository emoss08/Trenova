package agenttoolpolicy_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
)

// awaitingPreviewer is the write tools that do not yet say, record by
// record, what they would do. It only shrinks: a tool that gains a Preview
// leaves it, and a new write tool is never added to it.
var awaitingPreviewer = []string{
	"acknowledge_carrier_intel_event",
	"add_dashboard_tile",
	"add_home_widget",
	"approve_worker_pto",
	"arrange_home_layout",
	"assign_move",
	"attach_document_to_shipment",
	"check_accounting_connection",
	"clear_accounting_mapping",
	"create_accounting_reference_record",
	"create_dashboard",
	"create_location",
	"create_report",
	"create_shipment",
	"create_table_change_alert",
	"dismiss_insight",
	"escalate_detention",
	"evaluate_service_failures",
	"flag_for_manual_review",
	"forget_memory",
	"fork_report",
	"link_inbound_message",
	"mark_inbound_message",
	"place_shipment_hold",
	"place_worker_dispatch_hold",
	"raise_exception",
	"record_stop_actual",
	"refresh_accounting_reference_data",
	"release_shipment_hold",
	"remember",
	"remove_home_widget",
	"resolve_bank_receipt_work_item",
	"resolve_carrier_intel_event",
	"save_table_view",
	"set_accounting_mapping",
	"transition_item_to_in_review",
	"update_report",
	"update_tractor_status",
	"update_trailer_status",
}

// A person approving a write is shown what it would do, and only a tool that
// previews itself can say that. Every write tool the system registers is a
// ToolPreviewer.
func TestEveryWriteToolPreviewsItself(t *testing.T) {
	t.Parallel()

	for _, tool := range buildRegistered(t).Actions {
		policy := tool.Policy()
		if policy.Kind != agent.ToolKindAction || policy.Effect != agent.ToolEffectChange {
			continue
		}

		_, previews := tool.(serviceports.ToolPreviewer)
		awaiting := slices.Contains(awaitingPreviewer, tool.Name())
		switch {
		case !previews && !awaiting:
			t.Errorf("%s writes but has no Preview; a person would approve it blind", tool.Name())
		case previews && awaiting:
			t.Errorf("%s previews itself now; remove it from awaitingPreviewer", tool.Name())
		}
	}
}

func TestAwaitingPreviewerNamesOnlyRegisteredWriteTools(t *testing.T) {
	t.Parallel()

	registered := make(map[string]struct{})
	for _, tool := range buildRegistered(t).Actions {
		registered[tool.Name()] = struct{}{}
	}
	for _, name := range awaitingPreviewer {
		_, ok := registered[name]
		assert.True(t, ok, "%s is not a registered write tool", name)
	}
}
