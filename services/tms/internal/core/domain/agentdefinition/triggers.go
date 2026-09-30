package agentdefinition

import "github.com/emoss08/trenova/internal/core/domain/agent"

func (t Template) StarterTrigger() TriggerMode {
	switch t {
	case TemplateBillingException,
		TemplateShipmentIntake,
		TemplateCashApplication,
		TemplateDetentionDesk,
		TemplateCredentialDesk,
		TemplateCustomerUpdateDesk,
		TemplateCarrierRiskDesk,
		TemplateIntakeDesk,
		TemplateLoadEntryCheck,
		TemplateServiceFailureDesk,
		TemplateInsightAnalyst,
		TemplateEDIDesk,
		TemplateBooksKeeper,
		// The dispatch sweep now raises a move that is close enough to its
		// start to matter, so coverage is answered per move, with the move
		// as the run's subject, rather than by re-planning the board every
		// half hour and hoping a person reads the report.
		TemplateDispatchAssignment:
		return TriggerEvent
	case TemplateLoadMonitor:
		return TriggerScheduled
	default:
		return TriggerChat
	}
}

func (t Template) StarterEvents() []agent.EventKind {
	switch t {
	case TemplateBillingException:
		return []agent.EventKind{
			agent.EventBillingQueueItemException,
			agent.EventBillingQueueItemOnHold,
			agent.EventAccountingConnectionDegraded,
		}
	case TemplateShipmentIntake:
		return []agent.EventKind{agent.EventDocumentExtracted}
	case TemplateCashApplication:
		return []agent.EventKind{agent.EventBankReceiptException}
	case TemplateDispatchAssignment:
		return []agent.EventKind{
			agent.EventShipmentMoveCoverageAtRisk,
			agent.EventShipmentMoveUnassigned,
		}
	case TemplateDetentionDesk:
		return []agent.EventKind{
			agent.EventDetentionOccurrenceOpened,
			agent.EventDetentionNoticeDue,
		}
	case TemplateCredentialDesk:
		return []agent.EventKind{agent.EventWorkerCredentialExpiring}
	case TemplateCustomerUpdateDesk:
		return []agent.EventKind{
			agent.EventShipmentMoveArrived,
			agent.EventShipmentMoveDeparted,
		}
	case TemplateCarrierRiskDesk:
		return []agent.EventKind{agent.EventCarrierIntelEventOpened}
	case TemplateIntakeDesk:
		return []agent.EventKind{
			agent.EventInboundMessageClassified,
			agent.EventEDITenderReceived,
		}
	case TemplateLoadEntryCheck:
		return []agent.EventKind{agent.EventShipmentCreated}
	case TemplateServiceFailureDesk:
		return []agent.EventKind{agent.EventServiceFailureDetected}
	case TemplateInsightAnalyst:
		return []agent.EventKind{agent.EventInsightDetected}
	case TemplateEDIDesk:
		return []agent.EventKind{agent.EventEDIFileQuarantined}
	case TemplateBooksKeeper:
		return []agent.EventKind{
			agent.EventAccountingSyncFailed,
			agent.EventAccountingSyncBlocked,
			agent.EventAccountingConnectionDegraded,
			agent.EventAccountingPaymentProposed,
			agent.EventAccountingDriftDetected,
			agent.EventAccountingReconciliationDue,
		}
	default:
		return nil
	}
}

func (t Template) StarterCron() string {
	switch t {
	case TemplateLoadMonitor:
		return "*/15 * * * *"
	default:
		return ""
	}
}

func (t Template) StarterCeiling() agent.AutonomyTier {
	switch t {
	// The intake desk may earn running unattended, because an inbox the
	// organization set to handle mail without review is one it means to be
	// answered. The template only makes room: each inbox tool holds itself to
	// a proposal unless the message's own mailbox grants more. A tender that
	// arrives by email goes to shipment intake; one that arrives over EDI is
	// answered by a tool that always stops at a proposal a person decides.
	case TemplateIntakeDesk:
		return agent.TierAutoExecute
	case TemplateGeneralAssistant,
		TemplateCustomerAssistant,
		TemplateCustomerUpdateDesk,
		TemplateCarrierRiskDesk,
		TemplateLoadEntryCheck,
		TemplateInsightAnalyst,
		TemplateEDIDesk,
		TemplateFormulaAssistant,
		TemplateMasterDataSteward,
		TemplateWorkforceCoordinator,
		TemplateFuelTaxClerk,
		TemplateReportAnalyst:
		return agent.TierPropose
	default:
		return agent.TierActWithApproval
	}
}

// StarterShadow says whether an agent made from the template starts in shadow
// mode. The books keeper does: what it retries or maps reaches the accounting
// system, so an organization watches what it would do before letting it act.
func (t Template) StarterShadow() bool {
	return t == TemplateBooksKeeper
}

func (t Template) StarterDailyRunLimit() int {
	if t == TemplateInsightAnalyst {
		return insightAnalystDailyRuns
	}

	return 0
}
