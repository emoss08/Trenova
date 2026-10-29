package invoicepdfservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestResolveDeliveryProfileIncludesBillingControlWhenRequested(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	control := &billingcontrol.BillingControl{
		DefaultInvoiceTerms:  "Invoice terms",
		DefaultInvoiceFooter: "Invoice footer",
	}
	billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).Return(control, nil).Once()

	svc := &Service{
		billingRepo: billingRepo,
		rec:         invoiceaction.NewRecorder(invoiceaction.RecorderParams{Logger: zap.NewNop()}),
	}

	profile, err := svc.resolveDeliveryProfile(t.Context(), invoicedocument.DeliveryProfileParams{
		Entity:                &invoice.Invoice{},
		TenantInfo:            pagination.TenantInfo{OrgID: orgID},
		IncludeBillingControl: true,
	})

	require.NoError(t, err)
	require.Same(t, control, profile.BillingControl)
}

func TestResolveDeliveryProfileSkipsBillingControlWhenNotRequested(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
	svc := &Service{
		billingRepo: billingRepo,
		rec:         invoiceaction.NewRecorder(invoiceaction.RecorderParams{Logger: zap.NewNop()}),
	}

	profile, err := svc.resolveDeliveryProfile(t.Context(), invoicedocument.DeliveryProfileParams{
		Entity:     &invoice.Invoice{},
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	})

	require.NoError(t, err)
	require.Nil(t, profile.BillingControl)
}
