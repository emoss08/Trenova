package encryption

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiproviderservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/extensionsecrets"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringtransportservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/identity/app/authservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/mfaservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
)

func envelopeSources(includeObjects bool) *kernelports.EnvelopeSources {
	sources := &kernelports.EnvelopeSources{
		Columns: slices.Concat(
			accountingconnectionkit.EnvelopeColumns(),
			aiproviderservice.EnvelopeColumns(),
			extensionsecrets.EnvelopeColumns(),
			integrationservice.EnvelopeColumns(),
			factoringtransportservice.EnvelopeColumns(),
			inboundmessageservice.EnvelopeColumns(),
			ediprofilesecret.EnvelopeColumns(),
			authservice.EnvelopeColumns(),
			mfaservice.EnvelopeColumns(),
		),
	}
	if includeObjects {
		sources.Objects = slices.Concat(
			storedobject.EnvelopeObjects(),
			captureservice.EnvelopeObjects(),
		)
	}
	return sources
}
