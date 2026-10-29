package agentpreset

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
)

func StarterTrigger(t agenttemplate.Template) agentdefinition.TriggerMode {
	switch t {
	case agenttemplate.TemplateBillingException,
		agenttemplate.TemplateShipmentIntake,
		agenttemplate.TemplateCashApplication,
		agenttemplate.TemplateDetentionDesk,
		agenttemplate.TemplateCredentialDesk,
		agenttemplate.TemplateCustomerUpdateDesk,
		agenttemplate.TemplateCarrierRiskDesk,
		agenttemplate.TemplateIntakeDesk,
		agenttemplate.TemplateLoadEntryCheck,
		agenttemplate.TemplateServiceFailureDesk,
		agenttemplate.TemplateInsightAnalyst,
		agenttemplate.TemplateEDIDesk,
		agenttemplate.TemplateBooksKeeper,
		agenttemplate.TemplateFactoringClerk,
		// The dispatch sweep now raises a move that is close enough to its
		// start to matter, so coverage is answered per move, with the move
		// as the run's subject, rather than by re-planning the board every
		// half hour and hoping a person reads the report.
		agenttemplate.TemplateDispatchAssignment:
		return agentdefinition.TriggerEvent
	case agenttemplate.TemplateLoadMonitor:
		return agentdefinition.TriggerScheduled
	default:
		return agentdefinition.TriggerChat
	}
}

func StarterEvents(t agenttemplate.Template) []agenttrigger.EventKind {
	switch t {
	case agenttemplate.TemplateBillingException:
		return []agenttrigger.EventKind{
			agenttrigger.EventBillingQueueItemException,
			agenttrigger.EventBillingQueueItemOnHold,
			agenttrigger.EventAccountingConnectionDegraded,
		}
	case agenttemplate.TemplateShipmentIntake:
		return []agenttrigger.EventKind{agenttrigger.EventDocumentExtracted}
	case agenttemplate.TemplateCashApplication:
		return []agenttrigger.EventKind{agenttrigger.EventBankReceiptException}
	case agenttemplate.TemplateDispatchAssignment:
		return []agenttrigger.EventKind{
			agenttrigger.EventShipmentMoveCoverageAtRisk,
			agenttrigger.EventShipmentMoveUnassigned,
		}
	case agenttemplate.TemplateDetentionDesk:
		return []agenttrigger.EventKind{
			agenttrigger.EventDetentionOccurrenceOpened,
			agenttrigger.EventDetentionNoticeDue,
		}
	case agenttemplate.TemplateCredentialDesk:
		return []agenttrigger.EventKind{agenttrigger.EventWorkerCredentialExpiring}
	case agenttemplate.TemplateCustomerUpdateDesk:
		return []agenttrigger.EventKind{
			agenttrigger.EventShipmentMoveArrived,
			agenttrigger.EventShipmentMoveDeparted,
		}
	case agenttemplate.TemplateCarrierRiskDesk:
		return []agenttrigger.EventKind{agenttrigger.EventCarrierIntelEventOpened}
	case agenttemplate.TemplateIntakeDesk:
		return []agenttrigger.EventKind{
			agenttrigger.EventInboundMessageClassified,
			agenttrigger.EventEDITenderReceived,
		}
	case agenttemplate.TemplateLoadEntryCheck:
		return []agenttrigger.EventKind{agenttrigger.EventShipmentCreated}
	case agenttemplate.TemplateServiceFailureDesk:
		return []agenttrigger.EventKind{agenttrigger.EventServiceFailureDetected}
	case agenttemplate.TemplateInsightAnalyst:
		return []agenttrigger.EventKind{agenttrigger.EventInsightDetected}
	case agenttemplate.TemplateEDIDesk:
		return []agenttrigger.EventKind{agenttrigger.EventEDIFileQuarantined}
	case agenttemplate.TemplateFactoringClerk:
		return []agenttrigger.EventKind{
			agenttrigger.EventFactoringInvoiceReady,
			agenttrigger.EventFactoringNoaExpiring,
			agenttrigger.EventFactoringSubmissionSent,
			agenttrigger.EventFactoringSubmissionFailed,
			agenttrigger.EventFactoringSubmissionFunded,
			agenttrigger.EventFactoringChargebackDue,
			agenttrigger.EventFactoringRemittanceStaged,
		}
	case agenttemplate.TemplateBooksKeeper:
		return []agenttrigger.EventKind{
			agenttrigger.EventAccountingSyncFailed,
			agenttrigger.EventAccountingSyncBlocked,
			agenttrigger.EventAccountingConnectionDegraded,
			agenttrigger.EventAccountingPaymentProposed,
			agenttrigger.EventAccountingDriftDetected,
			agenttrigger.EventAccountingReconciliationDue,
		}
	default:
		return nil
	}
}

func StarterCron(t agenttemplate.Template) string {
	switch t {
	case agenttemplate.TemplateLoadMonitor:
		return "*/15 * * * *"
	default:
		return ""
	}
}

func StarterCeiling(t agenttemplate.Template) agent.AutonomyTier {
	switch t {
	// The intake desk may earn running unattended, because an inbox the
	// organization set to handle mail without review is one it means to be
	// answered. The template only makes room: each inbox tool holds itself to
	// a proposal unless the message's own mailbox grants more. A tender that
	// arrives by email goes to shipment intake; one that arrives over EDI is
	// answered by a tool that always stops at a proposal a person decides.
	case agenttemplate.TemplateIntakeDesk:
		return agent.TierAutoExecute
	case agenttemplate.TemplateGeneralAssistant,
		agenttemplate.TemplateCustomerAssistant,
		agenttemplate.TemplateCustomerUpdateDesk,
		agenttemplate.TemplateCarrierRiskDesk,
		agenttemplate.TemplateLoadEntryCheck,
		agenttemplate.TemplateInsightAnalyst,
		agenttemplate.TemplateEDIDesk,
		agenttemplate.TemplateFormulaAssistant,
		agenttemplate.TemplateMasterDataSteward,
		agenttemplate.TemplateWorkforceCoordinator,
		agenttemplate.TemplateFuelTaxClerk,
		agenttemplate.TemplateReportAnalyst:
		return agent.TierPropose
	default:
		return agent.TierActWithApproval
	}
}

// StarterShadow says whether an agent made from the template starts in shadow
// mode. The books keeper does: what it retries or maps reaches the accounting
// system, so an organization watches what it would do before letting it act.
func StarterShadow(t agenttemplate.Template) bool {
	return t == agenttemplate.TemplateBooksKeeper
}

func StarterDailyRunLimit(t agenttemplate.Template) int {
	if t == agenttemplate.TemplateInsightAnalyst {
		return insightAnalystDailyRuns
	}

	return 0
}
