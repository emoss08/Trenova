package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingissuetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/exceptiontools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
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
) agenttoolports.AgentTool {
	return billingissuetools.NewReleaseBillingQueueItemTool(billing, review)
}

func provideResolveBillingQueueIssueTool(
	review billingqueueports.BillingQueueReviewService,
) agenttoolports.AgentTool {
	return billingissuetools.NewResolveBillingQueueIssueTool(review)
}

func provideUndoBillingQueueIssueTool(
	review billingqueueports.BillingQueueReviewService,
) agenttoolports.AgentTool {
	return billingissuetools.NewUndoBillingQueueIssueTool(review)
}

func provideLogCollectionActivityTool(
	collections collectionports.Service,
) agenttoolports.AgentTool {
	return receivabletools.NewLogCollectionActivityTool(collections)
}

func provideAssignCollectorTool(collections collectionports.Service) agenttoolports.AgentTool {
	return receivabletools.NewAssignCollectorTool(collections)
}

func provideScheduleCollectionFollowUpTool(
	collections collectionports.Service,
) agenttoolports.AgentTool {
	return receivabletools.NewScheduleCollectionFollowUpTool(collections)
}

func provideSendPaymentReminderTool(
	reminders paymentreminderports.Service,
) agenttoolports.AgentTool {
	return receivabletools.NewSendPaymentReminderTool(reminders)
}

func provideSendCustomerStatementTool(
	statements customerstatementports.Service,
) agenttoolports.AgentTool {
	return receivabletools.NewSendCustomerStatementTool(statements)
}

func provideWakeExceptionTool(
	exceptions watchtowerports.ExceptionService,
) agenttoolports.AgentTool {
	return exceptiontools.NewWakeExceptionTool(exceptions)
}
