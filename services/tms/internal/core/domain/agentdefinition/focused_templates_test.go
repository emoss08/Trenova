package agentdefinition_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/require"
)

var withheldFromEveryTemplate = []string{
	"approve_rate_agreement",
	"reject_rate_agreement",
	"assign_pay_profile",
	"end_pay_assignment",
	"create_recurring_deduction",
	"update_recurring_deduction",
	"create_recurring_earning",
	"update_recurring_earning",
	"open_escrow_account",
	"update_escrow_account",
	"close_escrow_account",
	"draft_manual_journal",
	"revise_manual_journal_draft",
	"submit_manual_journal",
	"post_manual_journal",
	"cancel_manual_journal",
	"request_journal_reversal",
	"post_journal_reversal",
	"cancel_journal_reversal",
	"open_fiscal_period",
	"unlock_fiscal_period",
	"reopen_fiscal_period",
	"request_accounting_backfill",
	"change_accounting_backfill",
	"clear_accounting_mapping",
	"redate_accounting_sync",
	"forget_memory",
	"generate_payroll_export",
	"void_payroll_export",
	"review_driver_expense",
	"save_table_view",
	"add_home_widget",
	"remove_home_widget",
	"arrange_home_layout",
}

func TestTemplates_TheFocusedClerksTalkToAPersonAndOnlyPropose(t *testing.T) {
	t.Parallel()

	cases := map[agentdefinition.Template]struct {
		dataAccess agentdefinition.DataAccessCeiling
		icon       string
	}{
		agentdefinition.TemplateMasterDataSteward: {
			dataAccess: agentdefinition.DataAccessInternal,
			icon:       agentdefinition.IconFile,
		},
		agentdefinition.TemplateWorkforceCoordinator: {
			dataAccess: agentdefinition.DataAccessRestricted,
			icon:       agentdefinition.IconClipboard,
		},
		agentdefinition.TemplateFuelTaxClerk: {
			dataAccess: agentdefinition.DataAccessInternal,
			icon:       agentdefinition.IconGauge,
		},
	}

	for template, want := range cases {
		require.Truef(t, template.IsValid(), "%s", template)
		require.Containsf(t, agentdefinition.AllTemplates(), template, "%s", template)
		require.Equal(t, agentdefinition.TriggerChat, template.StarterTrigger(), template.Label())
		require.Empty(t, template.StarterEvents(), template.Label())
		require.Empty(t, template.StarterCron(), template.Label())
		require.Equal(t, agent.TierPropose, template.StarterCeiling(), template.Label())
		require.Equal(t, want.dataAccess, template.StarterDataAccess(), template.Label())
		require.False(t, template.StarterShadow(), template.Label())
		require.Zero(t, template.StarterDailyRunLimit(), template.Label())
		require.NotEmpty(t, template.Description(), template.Label())
		require.NotEmpty(t, template.StarterInstructions(), template.Label())
		require.Equal(t,
			want.icon,
			(&agentdefinition.Definition{Template: template}).ChosenIcon(),
			template.Label(),
		)
	}

	require.Equal(t, "Master data steward", agentdefinition.TemplateMasterDataSteward.Label())
	require.Equal(t,
		"Workforce coordinator", agentdefinition.TemplateWorkforceCoordinator.Label(),
	)
	require.Equal(t, "Fuel and IFTA clerk", agentdefinition.TemplateFuelTaxClerk.Label())
}

func TestTemplates_TheMasterDataStewardKeepsTheRecordsEveryoneWorksFrom(t *testing.T) {
	t.Parallel()

	tools := agentdefinition.TemplateMasterDataSteward.StarterTools()
	for _, tool := range []string{
		"create_carrier", "update_carrier", "update_carrier_status",
		"create_customer", "update_customer", "update_customer_status",
		"create_commodity", "update_commodity", "update_commodity_status",
		"create_hazardous_material", "update_hazardous_material",
		"update_hazardous_material_status",
		"create_location", "update_location", "update_location_status",
		"create_tractor", "update_tractor", "locate_tractor",
		"create_trailer", "update_trailer", "locate_trailer",
		"delete_documents", "restore_document_version",
		"create_table_change_alert", "update_table_change_alert",
		"set_table_change_alert_status", "delete_table_change_alert",
		"dismiss_watchtower_item", "file_capture_items", "discard_capture_item",
		"discard_capture_batch",
		"list_capture_batches", "list_equipment_manufacturers", "list_table_change_alerts",
		"list_watchtower_items",
	} {
		require.Containsf(t, tools, tool, "the master data steward needs %s", tool)
	}

	for _, tool := range []string{"assign_move", "post_invoice", "draft_rate_agreement"} {
		require.NotContainsf(t, tools, tool, "the master data steward must not hold %s", tool)
	}
}

func TestTemplates_TheWorkforceCoordinatorKeepsWorkerRecordsAndNeverPay(t *testing.T) {
	t.Parallel()

	tools := agentdefinition.TemplateWorkforceCoordinator.StarterTools()
	for _, tool := range []string{
		"get_worker",
		"request_worker_pto", "update_worker_pto", "approve_worker_pto", "reject_worker_pto",
		"cancel_worker_pto", "adjust_worker_pto_balance",
		"record_worker_injury", "update_worker_injury", "delete_worker_injury",
		"open_leave_case", "update_leave_case", "close_leave_case",
		"record_leave_day", "update_leave_day", "delete_leave_day",
		"start_performance_review", "draft_performance_review", "delete_performance_review",
		"give_worker_recognition", "delete_worker_recognition",
		"update_worker_safety_event", "delete_worker_safety_event",
		"update_safety_violation", "delete_safety_violation",
		"run_dot_random_draw", "finalize_dot_random_draw", "cancel_dot_random_draw",
		"cancel_dot_test",
		"start_worker_checklist", "cancel_worker_checklist",
		"close_worker_training", "archive_worker_credential",
		"record_shipment_permit", "update_shipment_permit",
	} {
		require.Containsf(t, tools, tool, "the workforce coordinator needs %s", tool)
	}

	for _, tool := range []string{
		"list_driver_settlements",
		"resolve_settlement_dispute",
		"list_payroll_exports",
		"list_driver_expenses",
	} {
		require.NotContainsf(t, tools, tool, "the workforce coordinator must not hold %s", tool)
	}
}

func TestTemplates_TheFuelTaxClerkKeepsTheFuelTaxRecord(t *testing.T) {
	t.Parallel()

	tools := agentdefinition.TemplateFuelTaxClerk.StarterTools()
	for _, tool := range []string{
		"list_fuel_purchases", "record_fuel_purchase", "correct_fuel_purchase",
		"delete_fuel_purchase",
		"list_fuel_cards", "assign_fuel_card",
		"list_fuel_purchase_imports", "commit_fuel_purchase_import",
		"resolve_fuel_purchase_import_rows", "discard_fuel_purchase_import",
		"list_ifta_returns", "generate_ifta_return", "recompute_ifta_return",
		"amend_ifta_return", "delete_ifta_return",
		"list_ifta_mileage_entries", "record_ifta_mileage_entry",
		"correct_ifta_mileage_entry", "delete_ifta_mileage_entry",
		"recalculate_move_jurisdiction_miles", "backfill_jurisdiction_miles",
		"list_ifta_jurisdictions",
		"list_fuel_index_prices", "record_fuel_index_price", "correct_fuel_index_price",
	} {
		require.Containsf(t, tools, tool, "the fuel and IFTA clerk needs %s", tool)
	}
}

func TestTemplates_WithholdWhatNoAgentShouldHold(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		tools := template.StarterTools()
		for _, tool := range withheldFromEveryTemplate {
			require.NotContainsf(t, tools, tool, "%s must not hold %s", template.Label(), tool)
		}
	}
}

func TestTemplates_TheExistingDesksPickUpTheWritesTheirWorkNeeds(t *testing.T) {
	t.Parallel()

	require.Contains(t,
		agentdefinition.TemplateDispatchAssistant.StarterTools(), "update_shipment",
	)
	require.Contains(t,
		agentdefinition.TemplateLoadMonitor.StarterTools(), "record_stop_actual",
	)
	require.NotContains(t,
		agentdefinition.TemplateShipmentIntake.StarterTools(), "cancel_shipment",
	)
}
