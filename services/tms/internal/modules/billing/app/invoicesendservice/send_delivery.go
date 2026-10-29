package invoicesendservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) resolveDeliveryProfile(
	ctx context.Context,
	params invoicedocument.DeliveryProfileParams,
) (*invoicedocument.DeliveryProfile, error) {
	return invoicedocument.ResolveDeliveryProfile(ctx, &invoicedocument.DeliveryProfileRepos{
		CustomerRepo:     s.customerRepo,
		ShipmentRepo:     s.shipmentRepo,
		BillingRepo:      s.billingRepo,
		OrganizationRepo: s.organizationRepo,
		FactoringRepo:    s.factoringRepo,
	}, params)
}

func (s *Service) resolveDeliveryOrganization(
	ctx context.Context,
	deliveryProfile *invoicedocument.DeliveryProfile,
	tenantInfo pagination.TenantInfo,
) error {
	return invoicedocument.ResolveDeliveryOrganization(
		ctx,
		s.organizationRepo,
		deliveryProfile,
		tenantInfo,
	)
}
