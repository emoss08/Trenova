package agentpreset

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentinstructions"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/shared/pulid"
)

var deskAreas = map[agenttemplate.Template][]string{
	agenttemplate.TemplateDispatchAssistant: {
		"shipment", "dispatch", "order", "equipment", "carrier", "location", "edi", "rate",
		"detention", "exception", "document", "insight", "masterdata",
	},
	agenttemplate.TemplateBillingAssistant: {
		"billing", "invoice", "receivable", "customer", "factoring", "rate", "document", "insight",
	},
	agenttemplate.TemplateComplianceAssistant: {
		"worker", "equipment", "document", "insight",
	},
}

var deskPins = map[agenttemplate.Template][]string{
	agenttemplate.TemplateDispatchAssistant: {
		"dispatch_get_board", "dispatch_plan", "dispatch_rank_candidates", "dispatch_assign_move",
		"shipment_add_comment",
	},
	agenttemplate.TemplateBillingAssistant: {
		"billing_approve_queue_item", "invoice_post", "invoice_send",
	},
	agenttemplate.TemplateComplianceAssistant: {
		"worker_request_credential_renewal", "worker_place_dispatch_hold",
	},
}

var withheldFromDesks = []string{
	"rate_approve_agreement",
	"rate_reject_agreement",
	"settlement_assign_pay_profile",
	"settlement_end_pay_assignment",
	"settlement_create_recurring_deduction",
	"settlement_update_recurring_deduction",
	"settlement_create_recurring_earning",
	"settlement_update_recurring_earning",
	"settlement_open_escrow_account",
	"settlement_update_escrow_account",
	"settlement_close_escrow_account",
	"ledger_create_manual_journal",
	"ledger_update_manual_journal",
	"ledger_submit_manual_journal",
	"ledger_post_manual_journal",
	"ledger_cancel_manual_journal",
	"ledger_create_journal_reversal",
	"ledger_post_journal_reversal",
	"ledger_cancel_journal_reversal",
	"ledger_open_fiscal_period",
	"ledger_unlock_fiscal_period",
	"ledger_reopen_fiscal_period",
	"accounting_request_backfill",
	"accounting_update_backfill",
	"accounting_clear_mapping",
	"accounting_redate_sync_record",
	"agent_retire_memory",
	"settlement_generate_payroll_export",
	"settlement_void_payroll_export",
	"settlement_review_driver_expense",
	"report_save_table_view",
	"home_add_widget",
	"home_remove_widget",
	"home_arrange_layout",
	"shipment_cancel",
}

func WithheldFromDesks() []string { return slices.Clone(withheldFromDesks) }

func DefaultDesks() []agenttemplate.Template {
	return []agenttemplate.Template{
		agenttemplate.TemplateUniversal,
		agenttemplate.TemplateDispatchAssistant,
		agenttemplate.TemplateBillingAssistant,
		agenttemplate.TemplateComplianceAssistant,
	}
}

func ScopeFor(template agenttemplate.Template) agentdefinition.Scope {
	if template == agenttemplate.TemplateUniversal {
		return agentdefinition.Scope{}
	}
	if areas, ok := deskAreas[template]; ok {
		return agentdefinition.Scope{
			Areas:            slices.Clone(areas),
			DenyCapabilities: slices.Clone(withheldFromDesks),
		}
	}
	tools := StarterTools(template)
	capabilities := make([]string, 0, len(tools))
	for _, tool := range tools {
		id := agentcapability.CapabilityIDForTool(tool)
		if !slices.Contains(capabilities, id) {
			capabilities = append(capabilities, id)
		}
	}

	return agentdefinition.Scope{Capabilities: capabilities}
}

func PinnedFor(template agenttemplate.Template) []string {
	if pins, ok := deskPins[template]; ok {
		return slices.Clone(pins)
	}
	standing := StandingTools(template)
	if len(standing) > agentdefinition.MaxPinnedTools {
		standing = standing[:agentdefinition.MaxPinnedTools]
	}

	return standing
}

var deskKeys = map[agenttemplate.Template]string{
	agenttemplate.TemplateUniversal:           "universal",
	agenttemplate.TemplateDispatchAssistant:   "dispatch_desk",
	agenttemplate.TemplateBillingAssistant:    "billing_desk",
	agenttemplate.TemplateComplianceAssistant: "safety_desk",
}

const UniversalAgentKey = "universal"

func DeskKey(template agenttemplate.Template) string { return deskKeys[template] }

func NewDesk(
	template agenttemplate.Template,
	orgID, buID pulid.ID,
) *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		Name:              template.Label(),
		Description:       template.Description(),
		Template:          template,
		Icon:              agentdefinition.TemplateIcon(template),
		Instructions:      agentinstructions.StarterInstructions(template),
		Scope:             ScopeFor(template),
		PinnedTools:       PinnedFor(template),
		AutonomyCeiling:   StarterCeiling(template),
		DataAccessCeiling: StarterDataAccess(template),
		TriggerMode:       agentdefinition.TriggerChat,
		OutputMode:        agentdefinition.OutputConversational,
		SystemKey:         DeskKey(template),
		AccessMode:        agentdefinition.AccessEveryone,
		ContextProviders:  agentdefinition.AllContextProviders(),
		Enabled:           true,
	}
	definition.ApplyDefaults()

	return definition
}
