//go:build integration

package edimessagerepository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument/edidocumentdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile/ediinboundfiledb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func seededDevelopmentDB(t *testing.T) (context.Context, *keel.DB) {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	return ctx, db
}

func tenantWithoutPartners(t *testing.T, ctx context.Context, db *keel.DB) editest.Tenant {
	t.Helper()

	var orgID, buID string
	require.NoError(t, db.Pool().QueryRow(ctx, `SELECT id, business_unit_id FROM organizations
		WHERE NOT EXISTS (SELECT 1 FROM edi_partners ep WHERE ep.organization_id = organizations.id)
		ORDER BY id LIMIT 1`).Scan(&orgID, &buID))

	return editest.TenantOf(
		ctx,
		pagination.TenantInfo{OrgID: pulid.ID(orgID), BuID: pulid.ID(buID)},
	)
}

func findDocumentType(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
) *edidocument.EDIDocumentType {
	t.Helper()

	dt := edidocumentdb.EDIDocumentTypes
	found, err := dt.Where(dt.TransactionSet.Eq(set), dt.Direction.Eq(direction)).First(ctx, db)
	require.NoError(t, err, "document type %s %s", set, direction)

	return found
}

func create(ctx context.Context, db *keel.DB, row any) error {
	switch r := row.(type) {
	case *edi.EDIPartner:
		return edidb.EDIPartners.Create(ctx, db, r)
	case *edi.EDICommunicationProfile:
		return edidb.EDICommunicationProfiles.Create(ctx, db, r)
	case *edimessage.EDIMessage:
		return edimessagedb.EDIMessages.Create(ctx, db, r)
	case *ediinboundfile.EDIInboundFile:
		return ediinboundfiledb.EDIInboundFiles.Create(ctx, db, r)
	case *editransfer.EDITransfer:
		return editransferdb.EDITransfers.Create(ctx, db, r)
	case *billingqueue.BillingQueueItem:
		return billingqueuedb.BillingQueueItems.Create(ctx, db, r)
	case *invoice.Invoice:
		return invoicedb.Invoices.Create(ctx, db, r)
	case *carrier.CarrierEDIChannel:
		return carrierdb.CarrierEDIChannels.Create(ctx, db, r)
	default:
		return fmt.Errorf("no keel table for %T", row)
	}
}
