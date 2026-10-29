package invoicedocument

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceassembly"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/mailtext"
)

// deliveryProfileRepos is the repository set profile resolution needs.
//
// It exists so the context builder can resolve a profile without taking a
// dependency on *Service, which depends on the template resolver and would
// therefore close an fx cycle.
type DeliveryProfileRepos struct {
	CustomerRepo     customerports.CustomerRepository
	ShipmentRepo     shipmentports.ShipmentRepository
	BillingRepo      tenantports.BillingControlRepository
	OrganizationRepo tenantports.OrganizationRepository
	FactoringRepo    factoringcompanyports.FactoringCompanyRepository
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
	shipmentRepo shipmentports.ShipmentRepository,
	result *DeliveryProfile,
	params DeliveryProfileParams,
) error {
	legIDs := params.Entity.LegShipmentIDs()

	switch len(legIDs) {
	case 0:
		return nil

	case 1:
		shp, err := shipmentRepo.GetByID(
			ctx,
			invoiceassembly.ExpandedShipmentByIDRequest(legIDs[0], params.TenantInfo),
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
			&shipmentports.ListShipmentSummariesRequest{
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

func ResolveDeliveryProfile(
	ctx context.Context,
	repos *DeliveryProfileRepos,
	params DeliveryProfileParams,
) (*DeliveryProfile, error) {
	result := &DeliveryProfile{}
	entity := params.Entity
	if entity == nil {
		return result, nil
	}

	applyLoadedParties(result, entity, params)
	if err := resolveDeliveryCustomer(ctx, repos.CustomerRepo, result, params); err != nil {
		return nil, err
	}
	if params.IncludeShipmentDetails && repos.ShipmentRepo != nil {
		if err := resolveDeliveryShipments(ctx, repos.ShipmentRepo, result, params); err != nil {
			return nil, err
		}
	}
	if err := resolveDeliveryBillingControl(ctx, repos.BillingRepo, result, params); err != nil {
		return nil, err
	}
	factor, err := resolveFactor(
		ctx,
		repos.FactoringRepo,
		entity,
		result.Customer,
		params.TenantInfo,
	)
	if err != nil {
		return nil, err
	}
	result.Factor = factor
	return result, nil
}

func applyLoadedParties(
	result *DeliveryProfile,
	entity *invoice.Invoice,
	params DeliveryProfileParams,
) {
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
}

func resolveDeliveryCustomer(
	ctx context.Context,
	customerRepo customerports.CustomerRepository,
	result *DeliveryProfile,
	params DeliveryProfileParams,
) error {
	if !params.IncludeCustomer || customerRepo == nil || params.Entity.CustomerID.IsNil() {
		return nil
	}
	cus, err := customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         params.Entity.CustomerID,
		TenantInfo: params.TenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeState:          params.IncludeCustomerState,
			IncludeBillingProfile: params.IncludeCustomerBillingProfile,
			IncludeEmailProfile:   params.IncludeCustomerEmailProfile,
		},
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}
	if cus == nil {
		return nil
	}
	result.Customer = cus
	if params.IncludeCustomerEmailProfile {
		result.Email = cus.EmailProfile
	}
	if cus.Organization != nil {
		result.Organization = cus.Organization
	}

	return nil
}

func resolveDeliveryBillingControl(
	ctx context.Context,
	billingRepo tenantports.BillingControlRepository,
	result *DeliveryProfile,
	params DeliveryProfileParams,
) error {
	if !params.IncludeBillingControl || billingRepo == nil {
		return nil
	}
	control, err := billingRepo.GetByOrgID(ctx, params.TenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}
	if control != nil {
		result.BillingControl = control
	}

	return nil
}

func ResolveDeliveryOrganization(
	ctx context.Context,
	organizationRepo tenantports.OrganizationRepository,
	deliveryProfile *DeliveryProfile,
	tenantInfo pagination.TenantInfo,
) error {
	if deliveryProfile == nil || deliveryProfile.Organization != nil || organizationRepo == nil {
		return nil
	}
	org, err := organizationRepo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
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

func ResolveRecipients(
	entity *invoice.Invoice,
	emailProfile *customer.CustomerEmailProfile,
) invoiceports.InvoiceSendRecipients {
	recipients := invoiceports.InvoiceSendRecipients{
		To:  mailtext.NormalizeEmailList(entity.EmailToSnapshot),
		CC:  mailtext.NormalizeEmailList(entity.EmailCCSnapshot),
		BCC: mailtext.NormalizeEmailList(entity.EmailBCCSnapshot),
	}
	if len(recipients.To) > 0 {
		return recipients
	}
	if emailProfile == nil {
		return recipients
	}
	recipients.To = mailtext.SplitEmailList(emailProfile.ToRecipients)
	recipients.CC = mailtext.SplitEmailList(emailProfile.CCRecipients)
	recipients.BCC = mailtext.SplitEmailList(emailProfile.BCCRecipients)
	return recipients
}
