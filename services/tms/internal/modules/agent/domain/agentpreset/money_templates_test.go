package agentpreset_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentinstructions"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmark"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/stretchr/testify/require"
)

/*
A template that fills the tool cap leaves an organization nothing to add. The
billing assistant sat at 63 of 64, so invoice_share and its candidate read were
registered and on no template: the only way to use them was to build an agent by
hand. Every starter keeps room for a handful of tools of the organization's own.
*/
const templateToolHeadroom = 8

func TestTemplates_LeaveRoomForAnOrganizationsOwnTools(t *testing.T) {
	t.Parallel()

	for _, template := range agenttemplate.AllTemplates() {
		require.LessOrEqualf(
			t,
			len(
				agentpreset.StarterTools(template),
			),
			agentdefinition.MaxTools-templateToolHeadroom,
			"%s holds %d tools, leaving fewer than %d of the %d an agent may hold",
			agenttemplate.Label(
				template,
			),
			len(agentpreset.StarterTools(template)),
			templateToolHeadroom,
			agentdefinition.MaxTools,
		)
	}
}

func TestTemplates_TheMoneyAssistantsTalkToAPersonAndActAsThem(t *testing.T) {
	t.Parallel()

	for _, template := range []agenttemplate.Template{
		agenttemplate.TemplateSettlementsClerk,
		agenttemplate.TemplateReceivables,
	} {
		require.Truef(t, template.IsValid(), "%s", template)
		require.Containsf(t, agenttemplate.AllTemplates(), template, "%s", template)
		require.Equal(
			t,
			agentdefinition.TriggerChat,
			agentpreset.StarterTrigger(template),
			agenttemplate.Label(template),
		)
		require.Empty(t, agentpreset.StarterEvents(template), agenttemplate.Label(template))
		require.Empty(t, agentpreset.StarterCron(template), agenttemplate.Label(template))
		require.Equal(
			t,
			agent.TierActWithApproval,
			agentpreset.StarterCeiling(template),
			agenttemplate.Label(template),
		)
		require.Equal(
			t,
			agentdefinition.DataAccessRestricted,
			agentpreset.StarterDataAccess(template),
			agenttemplate.Label(template),
		)
		require.False(t, agentpreset.StarterShadow(template), agenttemplate.Label(template))
		require.NotEmpty(t, agenttemplate.Description(template), agenttemplate.Label(template))
		require.NotEmpty(
			t,
			agentinstructions.StarterInstructions(template),
			agenttemplate.Label(template),
		)
	}
}

/*
The settlements clerk runs the pay period: it drafts settlements, sorts out
what is missing or held, adjusts, submits and proposes approving, posting and
paying, and matches carrier invoices. It reads how each driver is paid but
never changes it: a pay rate, a standing deduction or earning and escrow terms
are set apart from the settlements that apply them, so the one who processes
pay is not the one who sets it.
*/
func TestTemplates_TheSettlementsClerkRunsThePayPeriod(t *testing.T) {
	t.Parallel()

	tools := agentpreset.StarterTools(agenttemplate.TemplateSettlementsClerk)
	for _, tool := range []string{
		"worker_list",
		"carrier_list",
		"shipment_list",
		"settlement_list_driver",
		"settlement_get_driver",
		"settlement_list_pay_events",
		"settlement_get_driver_earnings",
		"settlement_get_dispute",
		"settlement_generate_driver",
		"settlement_generate_driver_batch",
		"settlement_attach_pay_events",
		"settlement_detach_pay_event",
		"settlement_hold_pay_event",
		"settlement_release_pay_event",
		"settlement_add_driver_adjustment",
		"settlement_remove_driver_adjustment",
		"settlement_recalculate_driver",
		"settlement_submit_driver",
		"settlement_approve_driver",
		"settlement_reject_driver",
		"settlement_post_driver",
		"settlement_record_driver_payment",
		"settlement_void_driver",
		"settlement_pay_now",
		"settlement_list_carrier",
		"settlement_get_carrier",
		"settlement_generate_carrier_batch",
		"settlement_add_carrier_adjustment",
		"settlement_remove_carrier_adjustment",
		"settlement_recalculate_carrier",
		"settlement_submit_carrier",
		"settlement_approve_carrier",
		"settlement_reject_carrier",
		"settlement_post_carrier",
		"settlement_record_carrier_payment",
		"settlement_void_carrier",
		"carrier_list_invoice_matches",
		"edi_list_carrier_invoices",
		"carrier_create_invoice_match",
		"carrier_accept_invoice_match",
		"carrier_accept_invoice_match_with_variance",
		"carrier_reject_invoice_match",
		"edi_link_carrier_invoice",
		"settlement_list_advances",
		"settlement_issue_advance",
		"settlement_forgive_advance",
		"settlement_list_escrow_accounts",
		"settlement_adjust_escrow_account",
		"settlement_list_pay_codes",
		"settlement_list_pay_profiles",
		"settlement_list_pay_assignments",
		"settlement_list_recurring_deductions",
		"settlement_list_recurring_earnings",
	} {
		require.Containsf(t, tools, tool, "the settlements clerk needs %s", tool)
	}

	for _, tool := range []string{
		"settlement_assign_pay_profile",
		"settlement_end_pay_assignment",
		"settlement_create_recurring_deduction",
		"settlement_update_recurring_deduction",
		"settlement_create_recurring_earning",
		"settlement_update_recurring_earning",
		"settlement_open_escrow_account",
		"settlement_update_escrow_account",
		"settlement_close_escrow_account",
		"receivable_post_customer_payment",
		"invoice_post",
	} {
		require.NotContainsf(t, tools, tool, "the settlements clerk must not hold %s", tool)
	}

	for _, tool := range []string{
		"settlement_approve_driver",
		"settlement_post_driver",
		"settlement_approve_carrier",
		"settlement_post_carrier",
	} {
		require.Containsf(t, tools, tool,
			"the settlements clerk approves and posts through %s, which takes one or many", tool)
	}
}

/*
Receivables works an invoice once it is in the customer's hands: what they
owe, what they paid, what they dispute and what they owe late. Money that
arrives at the bank is matched by the cash application agent, so receivables
works cash already recorded and never posts a payment or matches a receipt.
*/
func TestTemplates_ReceivablesWorksWhatCustomersOwe(t *testing.T) {
	t.Parallel()

	tools := agentpreset.StarterTools(agenttemplate.TemplateReceivables)
	for _, tool := range []string{
		"customer_list",
		"customer_get",
		"invoice_list",
		"invoice_get",
		"receivable_get_aging",
		"receivable_list_open_items",
		"receivable_list_collection_worklist",
		"receivable_get_statement",
		"receivable_record_collection_activity",
		"receivable_assign_collector",
		"receivable_schedule_follow_up",
		"receivable_send_payment_reminder",
		"receivable_send_statement",
		"receivable_list_customer_payments",
		"receivable_apply_customer_payment",
		"receivable_reverse_customer_payment",
		"receivable_list_credit_memo_applications",
		"receivable_apply_credit_memo",
		"receivable_unapply_credit_memo",
		"invoice_list_disputes",
		"invoice_open_dispute",
		"invoice_resolve_dispute",
		"invoice_withdraw_dispute",
		"invoice_list_adjustments",
		"receivable_assess_late_charges",
		"invoice_send",
		"invoice_list_share_candidates",
		"invoice_share",
	} {
		require.Containsf(t, tools, tool, "receivables needs %s", tool)
	}

	for _, tool := range []string{
		"receivable_post_customer_payment",
		"receivable_match_bank_receipt",
		"receivable_resolve_bank_receipt_work_item",
		"invoice_create",
		"invoice_void",
		"invoice_submit_adjustment",
		"billing_approve_queue_item",
	} {
		require.NotContainsf(t, tools, tool, "receivables must not hold %s", tool)
	}

	cash := agentpreset.StarterTools(agenttemplate.TemplateCashApplication)
	for _, tool := range []string{"receivable_match_bank_receipt", "receivable_post_customer_payment"} {
		require.Containsf(
			t,
			cash,
			tool,
			"the cash application agent still matches receipts with %s",
			tool,
		)
	}
}

// The billing assistant keeps the lifecycle up to an invoice in the customer's
// hands and its corrections; collections and cash move to receivables.
func TestTemplates_TheBillingAssistantHandsCollectionsToReceivables(t *testing.T) {
	t.Parallel()

	billing := agentpreset.StarterTools(agenttemplate.TemplateBillingAssistant)
	for _, tool := range []string{
		"billing_transfer",
		"billing_approve_queue_item",
		"invoice_post",
		"invoice_send",
		"invoice_create",
		"invoice_void",
		"invoice_create_memo",
		"invoice_submit_adjustment",
		"invoice_build_run",
		"invoice_commit_run",
		"invoice_bill_statement",
		"invoice_share",
		"invoice_list_share_candidates",
	} {
		require.Containsf(t, billing, tool, "the billing assistant keeps %s", tool)
	}

	for _, tool := range []string{
		"receivable_list_open_items",
		"receivable_list_customer_payments",
		"receivable_apply_customer_payment",
		"receivable_reverse_customer_payment",
		"receivable_list_credit_memo_applications",
		"receivable_apply_credit_memo",
		"receivable_unapply_credit_memo",
		"receivable_assess_late_charges",
		"invoice_list_disputes",
		"invoice_open_dispute",
		"invoice_resolve_dispute",
		"invoice_withdraw_dispute",
	} {
		require.NotContainsf(t, billing, tool, "%s is receivables work now", tool)
	}
}

func TestTemplates_EveryTemplateImpliesAnIcon(t *testing.T) {
	t.Parallel()

	for _, template := range agenttemplate.AllTemplates() {
		definition := &agentdefinition.Definition{Template: template}
		require.NotEmptyf(
			t,
			definition.ChosenIcon(),
			"%s implies no icon",
			agenttemplate.Label(template),
		)
	}

	clerk := &agentdefinition.Definition{Template: agenttemplate.TemplateSettlementsClerk}
	require.Equal(t, agentmark.IconBanknote, clerk.ChosenIcon())

	receivables := &agentdefinition.Definition{Template: agenttemplate.TemplateReceivables}
	require.Equal(t, agentmark.IconCoins, receivables.ChosenIcon())

	require.Contains(t, agentmark.KnownIcons(), agentmark.IconBanknote)
	require.Contains(t, agentmark.KnownIcons(), agentmark.IconCoins)
}

/*
The books keeper keeps the accounting system in step with what Trenova posts
and, once a period has ended, gets it ready to close. It confirms or turns down
the mappings Trenova proposed, proposes releasing what a review policy held,
and proposes locking and closing a period whose blockers are clear. It
proposes pausing sending while the connection fails and resuming once it
answers, creating an item, customer or vendor the books lack, and refreshing
what it read of the books. Drafting and posting journals, reversing entries, reopening a period and changing a
backfill are started by a person, so no unattended desk holds them.
*/
func TestTemplates_TheBooksKeeperKeepsTheBooksInStepAndReadyToClose(t *testing.T) {
	t.Parallel()

	tools := agentpreset.StarterTools(agenttemplate.TemplateBooksKeeper)
	for _, tool := range []string{
		"accounting_confirm_mapping",
		"accounting_reject_mapping",
		"accounting_release_sync_record",
		"ledger_list_fiscal_periods",
		"ledger_get_fiscal_close_blockers",
		"ledger_lock_fiscal_period",
		"ledger_close_fiscal_period",
		"accounting_pause_sync",
		"accounting_resume_sync",
		"accounting_create_reference_record",
		"accounting_refresh_reference_data",
	} {
		require.Containsf(t, tools, tool, "the books keeper needs %s", tool)
	}

	for _, tool := range []string{
		"ledger_create_manual_journal",
		"ledger_update_manual_journal",
		"ledger_submit_manual_journal",
		"ledger_cancel_manual_journal",
		"ledger_post_manual_journal",
		"ledger_create_journal_reversal",
		"ledger_cancel_journal_reversal",
		"ledger_post_journal_reversal",
		"ledger_reopen_fiscal_period",
		"ledger_unlock_fiscal_period",
		"ledger_open_fiscal_period",
		"accounting_update_backfill",
	} {
		require.NotContainsf(t, tools, tool, "the books keeper must not hold %s", tool)
	}

	cash := agentpreset.StarterTools(agenttemplate.TemplateCashApplication)
	require.Contains(t, cash, "receivable_triage_bank_receipt_work_item",
		"the cash application agent hands a receipt it cannot place to a person")
}
