package factoringtools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/factoringledgertools"
)

func Providers() []any {
	return []any{
		NewDraftFactoringNoaTool,
		NewSendFactoringNoaTool,
		NewRecordFactoringNoaAcknowledgementTool,
		NewReleaseFactoringNoaTool,
		NewSetInvoiceFactoringOverrideTool,
		NewRequestMissingFactoringDocumentTool,
		factoringledgertools.NewSubmitInvoicesToFactorTool,
		factoringledgertools.NewRetryFactoringSubmissionTool,
		factoringledgertools.NewCancelFactoringSubmissionTool,
		factoringledgertools.NewRecordFactoringFundingTool,
		factoringledgertools.NewReleaseFactoringReserveTool,
		factoringledgertools.NewRecordFactoringChargebackTool,
		factoringledgertools.NewReverseFactoringChargebackTool,
		factoringledgertools.NewResolveFactoringRemittanceLineTool,
		factoringledgertools.NewRematchFactoringRemittanceTool,
		factoringledgertools.NewApplyFactoringRemittanceTool,
		factoringledgertools.NewDiscardFactoringRemittanceTool,
	}
}
