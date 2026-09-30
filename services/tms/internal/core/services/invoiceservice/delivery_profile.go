package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// deliveryProfileRepos is the repository set profile resolution needs.
//
// It exists so the context builder can resolve a profile without taking a
// dependency on *Service, which depends on the template resolver and would
// therefore close an fx cycle.
type deliveryProfileRepos struct {
	customerRepo     repositories.CustomerRepository
	shipmentRepo     repositories.ShipmentRepository
	billingRepo      repositories.BillingControlRepository
	organizationRepo repositories.OrganizationRepository
}

func (s *Service) resolveDeliveryProfile(
	ctx context.Context,
	params resolveDeliveryProfileParams,
) (*invoiceDeliveryProfile, error) {
	return resolveDeliveryProfileWith(ctx, deliveryProfileRepos{
		customerRepo:     s.customerRepo,
		shipmentRepo:     s.shipmentRepo,
		billingRepo:      s.billingRepo,
		organizationRepo: s.organizationRepo,
	}, params)
}

// resolveDeliveryShipments loads whichever shape of freight this invoice bills.
//
// The header ShipmentID is only set on a single-shipment invoice, so reading it
// alone left a grouped invoice with no shipment at all — which is how every
// grouped invoice went out with an empty Shipper, an empty Consignee and no
// commodity rows. The lines are the authority on what an invoice covers, and
// LegShipmentIDs falls back to the header for the single case, so both shapes
// resolve from one source.
//
// One shipment still loads in full, because that is what the email context and
// the freight detail need. Several load as flat summaries instead: a document
// listing forty shipments prints a line about each, and fetching forty whole
// shipments to render forty lines would be forty round trips for data it throws
// away.
func resolveDeliveryShipments(
	ctx context.Context,
	shipmentRepo repositories.ShipmentRepository,
	result *invoiceDeliveryProfile,
	params resolveDeliveryProfileParams,
) error {
	legIDs := params.Entity.LegShipmentIDs()

	switch len(legIDs) {
	case 0:
		return nil

	case 1:
		shp, err := shipmentRepo.GetByID(
			ctx,
			expandedShipmentByIDRequest(legIDs[0], params.TenantInfo),
		)
		if err != nil && !errortypes.IsNotFoundError(err) {
			return err
		}
		if shp != nil {
			result.Shipment = shp
		}
		return nil

	default:
		summaries, err := shipmentRepo.ListSummariesByIDs(
			ctx,
			&repositories.ListShipmentSummariesRequest{
				TenantInfo:  params.TenantInfo,
				ShipmentIDs: legIDs,
			},
		)
		if err != nil && !errortypes.IsNotFoundError(err) {
			return err
		}
		result.Shipments = summaries
		return nil
	}
}

func resolveDeliveryProfileWith(
	ctx context.Context,
	repos deliveryProfileRepos,
	params resolveDeliveryProfileParams,
) (*invoiceDeliveryProfile, error) {
	result := &invoiceDeliveryProfile{}
	entity := params.Entity
	if entity == nil {
		return result, nil
	}

	if entity.Customer != nil {
		result.Customer = entity.Customer
		if params.IncludeCustomerEmailProfile {
			result.Email = entity.Customer.EmailProfile
		}
		result.Organization = entity.Customer.Organization
	}
	if entity.Shipment != nil {
		result.Shipment = entity.Shipment
	}
	if params.IncludeCustomer && repos.customerRepo != nil && entity.CustomerID.IsNotNil() {
		cus, err := repos.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID:         entity.CustomerID,
			TenantInfo: params.TenantInfo,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeState:          params.IncludeCustomerState,
				IncludeBillingProfile: params.IncludeCustomerBillingProfile,
				IncludeEmailProfile:   params.IncludeCustomerEmailProfile,
			},
		})
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}
		if cus != nil {
			result.Customer = cus
			if params.IncludeCustomerEmailProfile {
				result.Email = cus.EmailProfile
			}
			if cus.Organization != nil {
				result.Organization = cus.Organization
			}
		}
	}
	if params.IncludeShipmentDetails && repos.shipmentRepo != nil {
		if err := resolveDeliveryShipments(ctx, repos.shipmentRepo, result, params); err != nil {
			return nil, err
		}
	}
	if params.IncludeBillingControl && repos.billingRepo != nil {
		control, err := repos.billingRepo.GetByOrgID(ctx, params.TenantInfo.OrgID)
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}
		if control != nil {
			result.BillingControl = control
		}
	}
	return result, nil
}

func basicShipmentByIDRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *repositories.GetShipmentByIDRequest {
	return shipmentByIDRequest(id, tenantInfo, false)
}

func expandedShipmentByIDRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *repositories.GetShipmentByIDRequest {
	return shipmentByIDRequest(id, tenantInfo, true)
}

func shipmentByIDRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	expandShipmentDetails bool,
) *repositories.GetShipmentByIDRequest {
	return &repositories.GetShipmentByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: expandShipmentDetails,
		},
	}
}

func (s *Service) resolveDeliveryOrganization(
	ctx context.Context,
	deliveryProfile *invoiceDeliveryProfile,
	tenantInfo pagination.TenantInfo,
) error {
	return resolveDeliveryOrganizationWith(ctx, s.organizationRepo, deliveryProfile, tenantInfo)
}

func resolveDeliveryOrganizationWith(
	ctx context.Context,
	organizationRepo repositories.OrganizationRepository,
	deliveryProfile *invoiceDeliveryProfile,
	tenantInfo pagination.TenantInfo,
) error {
	if deliveryProfile == nil || deliveryProfile.Organization != nil || organizationRepo == nil {
		return nil
	}
	org, err := organizationRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}
	if org != nil {
		deliveryProfile.Organization = org
	}
	return nil
}

func resolveRecipients(
	entity *invoice.Invoice,
	emailProfile *customer.CustomerEmailProfile,
) servicesports.InvoiceSendRecipients {
	recipients := servicesports.InvoiceSendRecipients{
		To:  stringutils.NormalizeEmailList(entity.EmailToSnapshot),
		CC:  stringutils.NormalizeEmailList(entity.EmailCCSnapshot),
		BCC: stringutils.NormalizeEmailList(entity.EmailBCCSnapshot),
	}
	if len(recipients.To) > 0 {
		return recipients
	}
	if emailProfile == nil {
		return recipients
	}
	recipients.To = stringutils.SplitEmailList(emailProfile.ToRecipients)
	recipients.CC = stringutils.SplitEmailList(emailProfile.CCRecipients)
	recipients.BCC = stringutils.SplitEmailList(emailProfile.BCCRecipients)
	return recipients
}
