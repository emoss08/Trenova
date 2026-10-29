package templateregistry

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/accounttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/detentiontemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/factoringtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/invoicetemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/portaltemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/rateconfirmationtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/reporttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/statementtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
	"github.com/emoss08/trenova/internal/modules/documents/domain/tendertemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/trackingtemplate"
)

func NewRegistry() *templatecatalog.Registry {
	return templatecatalog.New(
		invoicetemplate.RegisterBillingKinds,
		invoicetemplate.RegisterInvoiceShareKinds,
		invoicetemplate.RegisterPaymentReminderKinds,
		statementtemplate.RegisterCustomerStatementKinds,
		factoringtemplate.RegisterFactoringNoaKinds,
		factoringtemplate.RegisterFactoringScheduleKinds,
		detentiontemplate.RegisterDetentionKinds,
		rateconfirmationtemplate.RegisterRateConfirmationKinds,
		reporttemplate.RegisterReportingKinds,
		tendertemplate.RegisterTenderKinds,
		trackingtemplate.RegisterCarrierTrackingKinds,
		trackingtemplate.RegisterCustomerTrackingKinds,
		portaltemplate.RegisterPortalKinds,
		accounttemplate.RegisterAccountKinds,
		portaltemplate.RegisterAgentKinds,
		notificationtemplate.RegisterNotificationKinds,
	)
}
