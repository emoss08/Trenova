package agentdefinition_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/require"
)

/*
A template that fills the tool cap leaves an organization nothing to add. The
billing assistant sat at 63 of 64, so share_invoice and its candidate read were
registered and on no template: the only way to use them was to build an agent by
hand. Every starter keeps room for a handful of tools of the organization's own.
*/
const templateToolHeadroom = 8

func TestTemplates_LeaveRoomForAnOrganizationsOwnTools(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		require.LessOrEqualf(t,
			len(template.StarterTools()), agentdefinition.MaxTools-templateToolHeadroom,
			"%s holds %d tools, leaving fewer than %d of the %d an agent may hold",
			template.Label(), len(template.StarterTools()), templateToolHeadroom,
			agentdefinition.MaxTools,
		)
	}
}

func TestTemplates_TheMoneyAssistantsTalkToAPersonAndActAsThem(t *testing.T) {
	t.Parallel()

	for _, template := range []agentdefinition.Template{
		agentdefinition.TemplateSettlementsClerk,
		agentdefinition.TemplateReceivables,
	} {
		require.Truef(t, template.IsValid(), "%s", template)
		require.Containsf(t, agentdefinition.AllTemplates(), template, "%s", template)
		require.Equal(t, agentdefinition.TriggerChat, template.StarterTrigger(), template.Label())
		require.Empty(t, template.StarterEvents(), template.Label())
		require.Empty(t, template.StarterCron(), template.Label())
		require.Equal(t, agent.TierActWithApproval, template.StarterCeiling(), template.Label())
		require.Equal(t,
			agentdefinition.DataAccessRestricted, template.StarterDataAccess(), template.Label(),
		)
		require.False(t, template.StarterShadow(), template.Label())
		require.NotEmpty(t, template.Description(), template.Label())
		require.NotEmpty(t, template.StarterInstructions(), template.Label())
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

	tools := agentdefinition.TemplateSettlementsClerk.StarterTools()
	for _, tool := range []string{
		"search_worker",
		"list_carriers",
		"search_shipments",
		"list_driver_settlements",
		"get_driver_settlement",
		"list_driver_pay_events",
		"get_worker_earnings_summary",
		"get_settlement_dispute",
		"generate_driver_settlement",
		"generate_driver_settlement_batch",
		"attach_pay_events_to_settlement",
		"detach_pay_event_from_settlement",
		"hold_driver_pay_event",
		"release_driver_pay_event",
		"add_driver_settlement_adjustment",
		"remove_driver_settlement_adjustment",
		"recalculate_driver_settlement",
		"submit_driver_settlement",
		"approve_driver_settlement",
		"reject_driver_settlement",
		"post_driver_settlement",
		"record_driver_settlement_payment",
		"void_driver_settlement",
		"pay_driver_now",
		"list_carrier_settlements",
		"get_carrier_settlement",
		"generate_carrier_settlement_batch",
		"add_carrier_settlement_adjustment",
		"remove_carrier_settlement_adjustment",
		"recalculate_carrier_settlement",
		"submit_carrier_settlement",
		"approve_carrier_settlement",
		"reject_carrier_settlement",
		"post_carrier_settlement",
		"record_carrier_settlement_payment",
		"void_carrier_settlement",
		"list_carrier_invoice_matches",
		"list_edi_carrier_invoices",
		"create_carrier_invoice_match",
		"accept_carrier_invoice_match",
		"accept_carrier_invoice_match_with_variance",
		"reject_carrier_invoice_match",
		"link_edi_carrier_invoice_to_carrier",
		"list_pay_advances",
		"issue_pay_advance",
		"write_off_pay_advance",
		"list_escrow_accounts",
		"adjust_escrow_account",
		"list_pay_codes",
		"list_pay_profiles",
		"list_pay_assignments",
		"list_recurring_deductions",
		"list_recurring_earnings",
	} {
		require.Containsf(t, tools, tool, "the settlements clerk needs %s", tool)
	}

	for _, tool := range []string{
		"assign_pay_profile",
		"end_pay_assignment",
		"create_recurring_deduction",
		"update_recurring_deduction",
		"create_recurring_earning",
		"update_recurring_earning",
		"open_escrow_account",
		"update_escrow_account",
		"close_escrow_account",
		"post_customer_payment",
		"post_invoice",
	} {
		require.NotContainsf(t, tools, tool, "the settlements clerk must not hold %s", tool)
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

	tools := agentdefinition.TemplateReceivables.StarterTools()
	for _, tool := range []string{
		"list_customers",
		"get_customer",
		"list_invoices",
		"get_invoice",
		"get_ar_aging",
		"list_ar_open_items",
		"list_collections_worklist",
		"get_customer_statement",
		"list_customer_payments",
		"apply_customer_payment",
		"reverse_customer_payment",
		"list_credit_memo_applications",
		"apply_credit_memo",
		"unapply_credit_memo",
		"list_invoice_disputes",
		"open_invoice_dispute",
		"resolve_invoice_dispute",
		"withdraw_invoice_dispute",
		"list_invoice_adjustments",
		"assess_late_charges",
		"send_invoice",
		"list_invoice_share_candidates",
		"share_invoice",
	} {
		require.Containsf(t, tools, tool, "receivables needs %s", tool)
	}

	for _, tool := range []string{
		"post_customer_payment",
		"match_bank_receipt",
		"resolve_bank_receipt_work_item",
		"create_invoice",
		"void_invoice",
		"submit_invoice_adjustment",
		"approve_billing_queue_item",
	} {
		require.NotContainsf(t, tools, tool, "receivables must not hold %s", tool)
	}

	cash := agentdefinition.TemplateCashApplication.StarterTools()
	for _, tool := range []string{"match_bank_receipt", "post_customer_payment"} {
		require.Containsf(t, cash, tool, "the cash application agent still matches receipts with %s", tool)
	}
}

// The billing assistant keeps the lifecycle up to an invoice in the customer's
// hands and its corrections; collections and cash move to receivables.
func TestTemplates_TheBillingAssistantHandsCollectionsToReceivables(t *testing.T) {
	t.Parallel()

	billing := agentdefinition.TemplateBillingAssistant.StarterTools()
	for _, tool := range []string{
		"transfer_to_billing",
		"approve_billing_queue_item",
		"post_invoice",
		"send_invoice",
		"create_invoice",
		"void_invoice",
		"create_invoice_memo",
		"submit_invoice_adjustment",
		"build_invoice_run",
		"commit_invoice_run",
		"bill_statement_now",
		"share_invoice",
		"list_invoice_share_candidates",
	} {
		require.Containsf(t, billing, tool, "the billing assistant keeps %s", tool)
	}

	for _, tool := range []string{
		"list_ar_open_items",
		"list_customer_payments",
		"apply_customer_payment",
		"reverse_customer_payment",
		"list_credit_memo_applications",
		"apply_credit_memo",
		"unapply_credit_memo",
		"assess_late_charges",
		"list_invoice_disputes",
		"open_invoice_dispute",
		"resolve_invoice_dispute",
		"withdraw_invoice_dispute",
	} {
		require.NotContainsf(t, billing, tool, "%s is receivables work now", tool)
	}
}

func TestTemplates_EveryTemplateImpliesAnIcon(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		definition := &agentdefinition.Definition{Template: template}
		require.NotEmptyf(t, definition.ChosenIcon(), "%s implies no icon", template.Label())
	}

	clerk := &agentdefinition.Definition{Template: agentdefinition.TemplateSettlementsClerk}
	require.Equal(t, agentdefinition.IconBanknote, clerk.ChosenIcon())

	receivables := &agentdefinition.Definition{Template: agentdefinition.TemplateReceivables}
	require.Equal(t, agentdefinition.IconCoins, receivables.ChosenIcon())

	require.Contains(t, agentdefinition.KnownIcons(), agentdefinition.IconBanknote)
	require.Contains(t, agentdefinition.KnownIcons(), agentdefinition.IconCoins)
}
