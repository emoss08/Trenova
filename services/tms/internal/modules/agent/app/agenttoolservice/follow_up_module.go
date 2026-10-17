package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
)

func followUpToolProviders() []any {
	return []any{
		provideReleaseBillingQueueItemTool,
		provideResolveBillingQueueIssueTool,
		provideUndoBillingQueueIssueTool,
		movetools.ProvideIssueCarrierTrackingLinkTool,
		movetools.ProvideRevokeCarrierTrackingLinkTool,
		provideLogCollectionActivityTool,
		provideAssignCollectorTool,
		provideScheduleCollectionFollowUpTool,
		provideSendPaymentReminderTool,
		provideSendCustomerStatementTool,
		provideWakeExceptionTool,
	}
}

func provideReleaseBillingQueueItemTool(
	billing billingqueueports.BillingQueueService,
	review billingqueueports.BillingQueueReviewService,
) agentports.AgentTool {
	return billingtools.NewReleaseBillingQueueItemTool(billing, review)
}

func provideResolveBillingQueueIssueTool(
	review billingqueueports.BillingQueueReviewService,
) agentports.AgentTool {
	return billingtools.NewResolveBillingQueueIssueTool(review)
}

func provideUndoBillingQueueIssueTool(
	review billingqueueports.BillingQueueReviewService,
) agentports.AgentTool {
	return billingtools.NewUndoBillingQueueIssueTool(review)
}

func provideLogCollectionActivityTool(collections collectionports.Service) agentports.AgentTool {
	return receivabletools.NewLogCollectionActivityTool(collections)
}

func provideAssignCollectorTool(collections collectionports.Service) agentports.AgentTool {
	return receivabletools.NewAssignCollectorTool(collections)
}

func provideScheduleCollectionFollowUpTool(
	collections collectionports.Service,
) agentports.AgentTool {
	return receivabletools.NewScheduleCollectionFollowUpTool(collections)
}

func provideSendPaymentReminderTool(reminders paymentreminderports.Service) agentports.AgentTool {
	return receivabletools.NewSendPaymentReminderTool(reminders)
}

func provideSendCustomerStatementTool(
	statements customerstatementports.Service,
) agentports.AgentTool {
	return receivabletools.NewSendCustomerStatementTool(statements)
}

func provideWakeExceptionTool(exceptions watchtowerports.ExceptionService) agentports.AgentTool {
	return receivabletools.NewWakeExceptionTool(exceptions)
}
