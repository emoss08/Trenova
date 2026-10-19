package agentdefinition_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/stretchr/testify/require"
)

const (
	withheldRateReview = "the agent that drafts a rate agreement is never the one that " +
		"reviews it"
	withheldPaySetup = "pay setup is kept apart from the settlements that apply it, so the " +
		"one who processes pay is not the one who sets it"
	withheldPrivilegedAccounting = "privileged accounting starts from a person's request and " +
		"belongs to no unattended desk"
	withheldMemory = "a person retires a memory in AI control; an agent never drops what " +
		"it was told on its own judgement"
	withheldPayMovement = "it moves pay, which belongs to the settlements clerk, and that " +
		"template is at the tool cap"
	withheldPersonalInterface = "it changes only the caller's own screen, a person's " +
		"interface state that is no desk's job"
	withheldCancelShipment = "canceling a shipment is a person's call; shipment_uncancel " +
		"stays with the dispatch assistant"
)

var withheldFromEveryTemplate = map[string]string{
	"rate_approve_agreement":                withheldRateReview,
	"rate_reject_agreement":                 withheldRateReview,
	"settlement_assign_pay_profile":         withheldPaySetup,
	"settlement_end_pay_assignment":         withheldPaySetup,
	"settlement_create_recurring_deduction": withheldPaySetup,
	"settlement_update_recurring_deduction": withheldPaySetup,
	"settlement_create_recurring_earning":   withheldPaySetup,
	"settlement_update_recurring_earning":   withheldPaySetup,
	"settlement_open_escrow_account":        withheldPaySetup,
	"settlement_update_escrow_account":      withheldPaySetup,
	"settlement_close_escrow_account":       withheldPaySetup,
	"ledger_create_manual_journal":          withheldPrivilegedAccounting,
	"ledger_update_manual_journal":          withheldPrivilegedAccounting,
	"ledger_submit_manual_journal":          withheldPrivilegedAccounting,
	"ledger_post_manual_journal":            withheldPrivilegedAccounting,
	"ledger_cancel_manual_journal":          withheldPrivilegedAccounting,
	"ledger_create_journal_reversal":        withheldPrivilegedAccounting,
	"ledger_post_journal_reversal":          withheldPrivilegedAccounting,
	"ledger_cancel_journal_reversal":        withheldPrivilegedAccounting,
	"ledger_open_fiscal_period":             withheldPrivilegedAccounting,
	"ledger_unlock_fiscal_period":           withheldPrivilegedAccounting,
	"ledger_reopen_fiscal_period":           withheldPrivilegedAccounting,
	"accounting_request_backfill":           withheldPrivilegedAccounting,
	"accounting_update_backfill":            withheldPrivilegedAccounting,
	"accounting_clear_mapping":              withheldPrivilegedAccounting,
	"accounting_redate_sync_record":         withheldPrivilegedAccounting,
	"agent_retire_memory":                   withheldMemory,
	"settlement_generate_payroll_export":    withheldPayMovement,
	"settlement_void_payroll_export":        withheldPayMovement,
	"settlement_review_driver_expense":      withheldPayMovement,
	"report_save_table_view":                withheldPersonalInterface,
	"home_add_widget":                       withheldPersonalInterface,
	"home_remove_widget":                    withheldPersonalInterface,
	"home_arrange_layout":                   withheldPersonalInterface,
	"shipment_cancel":                       withheldCancelShipment,
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
		agentdefinition.TemplateReportAnalyst: {
			dataAccess: agentdefinition.DataAccessInternal,
			icon:       agentdefinition.IconCompass,
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
	require.Equal(t, "Report analyst", agentdefinition.TemplateReportAnalyst.Label())
}

func TestTemplates_TheMasterDataStewardKeepsTheRecordsEveryoneWorksFrom(t *testing.T) {
	t.Parallel()

	tools := agentdefinition.TemplateMasterDataSteward.StarterTools()
	for _, tool := range []string{
		"carrier_create", "carrier_update", "carrier_set_status",
		"customer_create", "customer_update", "customer_set_status",
		"masterdata_create_commodity", "masterdata_update_commodity", "masterdata_set_commodity_status",
		"masterdata_create_hazardous_material", "masterdata_update_hazardous_material",
		"masterdata_set_hazardous_material_status",
		"location_create", "location_update", "location_set_status",
		"equipment_create_tractor", "equipment_update_tractor", "equipment_set_tractor_location",
		"equipment_create_trailer", "equipment_update_trailer", "equipment_set_trailer_location",
		"document_delete", "document_restore_version",
		"report_create_table_change_alert", "report_update_table_change_alert",
		"report_set_table_change_alert_status", "report_delete_table_change_alert",
		"exception_dismiss_watchtower_item", "inbox_file_capture_item", "inbox_discard_capture_item",
		"inbox_discard_capture_batch",
		"inbox_list_capture_batches", "equipment_list_manufacturers", "report_list_table_change_alerts",
		"exception_list_watchtower_items",
	} {
		require.Containsf(t, tools, tool, "the master data steward needs %s", tool)
	}

	for _, tool := range []string{"dispatch_assign_move", "invoice_post", "rate_create_agreement"} {
		require.NotContainsf(t, tools, tool, "the master data steward must not hold %s", tool)
	}
}

func TestTemplates_TheWorkforceCoordinatorKeepsWorkerRecordsAndNeverPay(t *testing.T) {
	t.Parallel()

	tools := agentdefinition.TemplateWorkforceCoordinator.StarterTools()
	for _, tool := range []string{
		"worker_get",
		"worker_request_pto", "worker_update_pto", "worker_approve_pto", "worker_reject_pto",
		"worker_cancel_pto", "worker_adjust_pto_balance",
		"worker_record_injury", "worker_update_injury", "worker_delete_injury",
		"worker_open_leave_case", "worker_update_leave_case", "worker_close_leave_case",
		"worker_record_leave_day", "worker_update_leave_day", "worker_delete_leave_day",
		"worker_start_performance_review", "worker_update_performance_review", "worker_delete_performance_review",
		"worker_record_recognition", "worker_delete_recognition",
		"worker_update_safety_event", "worker_delete_safety_event",
		"worker_update_safety_violation", "worker_delete_safety_violation",
		"worker_run_dot_random_draw", "worker_finalize_dot_random_draw", "worker_cancel_dot_random_draw",
		"worker_cancel_dot_test",
		"worker_start_checklist", "worker_cancel_checklist",
		"worker_close_training", "worker_archive_credential",
		"shipment_record_permit", "shipment_update_permit",
	} {
		require.Containsf(t, tools, tool, "the workforce coordinator needs %s", tool)
	}

	for _, tool := range []string{
		"settlement_list_driver",
		"settlement_resolve_dispute",
		"settlement_list_payroll_exports",
		"settlement_list_driver_expenses",
	} {
		require.NotContainsf(t, tools, tool, "the workforce coordinator must not hold %s", tool)
	}
}

func TestTemplates_TheFuelTaxClerkKeepsTheFuelTaxRecord(t *testing.T) {
	t.Parallel()

	tools := agentdefinition.TemplateFuelTaxClerk.StarterTools()
	for _, tool := range []string{
		"fuel_list_purchases", "fuel_record_purchase", "fuel_correct_purchase",
		"fuel_delete_purchase",
		"fuel_list_cards", "fuel_assign_card",
		"fuel_list_purchase_imports", "fuel_commit_purchase_import",
		"fuel_resolve_purchase_import_rows", "fuel_discard_purchase_import",
		"fuel_list_ifta_returns", "fuel_generate_ifta_return", "fuel_recalculate_ifta_return",
		"fuel_amend_ifta_return", "fuel_delete_ifta_return",
		"fuel_list_ifta_mileage_entries", "fuel_record_ifta_mileage_entry",
		"fuel_correct_ifta_mileage_entry", "fuel_delete_ifta_mileage_entry",
		"fuel_recalculate_jurisdiction_miles", "fuel_backfill_jurisdiction_miles",
		"fuel_list_ifta_jurisdictions",
		"rate_list_fuel_index_prices", "rate_record_fuel_index_price", "rate_correct_fuel_index_price",
	} {
		require.Containsf(t, tools, tool, "the fuel and IFTA clerk needs %s", tool)
	}
}

func TestTemplates_TheReportAnalystBuildsRunsAndSchedulesReports(t *testing.T) {
	t.Parallel()

	tools := agentdefinition.TemplateReportAnalyst.StarterTools()
	for _, tool := range []string{
		"report_list", "report_get", "report_preview", "report_run", "report_get_run",
		"report_list_runs", "report_compare_runs", "report_cancel_run",
		"report_list_datasets", "report_get_dataset",
		"report_create", "report_update", "report_fork", "report_reset_fork", "report_delete",
		"report_list_schedules", "report_create_schedule", "report_update_schedule",
		"report_delete_schedule",
		"report_list_dashboards", "report_create_dashboard", "report_add_dashboard_tile", "report_delete_dashboard",
	} {
		require.Containsf(t, tools, tool, "the report analyst needs %s", tool)
	}

	for _, tool := range []string{"home_add_widget", "report_save_table_view", "customer_send_email"} {
		require.NotContainsf(t, tools, tool, "the report analyst must not hold %s", tool)
	}
}

func TestTemplates_WithholdWhatNoAgentShouldHold(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		tools := template.StarterTools()
		for tool, why := range withheldFromEveryTemplate {
			require.NotContainsf(t, tools, tool, "%s must not hold %s: %s",
				template.Label(), tool, why)
		}
	}
}

func TestTemplates_TheExistingDesksPickUpTheWritesTheirWorkNeeds(t *testing.T) {
	t.Parallel()

	require.Contains(t,
		agentdefinition.TemplateDispatchAssistant.StarterTools(), "shipment_update",
	)
	require.Contains(t,
		agentdefinition.TemplateDispatchAssistant.StarterTools(), "shipment_uncancel",
	)
	require.Contains(t,
		agentdefinition.TemplateLoadMonitor.StarterTools(), "dispatch_record_stop_actual",
	)
	require.NotContains(t,
		agentdefinition.TemplateShipmentIntake.StarterTools(), "shipment_cancel",
	)
}
