package templateregistry

import (
	"sync"

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

// defaultRegistry is the template catalog, built once and never changed.
var defaultRegistry = sync.OnceValue(NewRegistry)

// Default is the process's template catalog. fx provides it to services; code
// outside the graph (starters, reference seeds) reads it here, so the process
// holds one catalog however it is reached.
func Default() *templatecatalog.Registry { return defaultRegistry() }

func NewRegistry() *templatecatalog.Registry {
	return templatecatalog.New(
		invoicetemplate.RegisterBillingKinds,
		notificationtemplate.RegisterRecordShareKinds,
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
