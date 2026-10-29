package invoicememoservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceassembly"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type memoPlan struct {
	customer *customer.Customer
	control  *billingcontrol.BillingControl
	lines    []*invoice.InvoiceLine
}

func (s *Service) planMemo(
	ctx context.Context,
	req *invoiceports.CreateMemoRequest,
) (*memoPlan, error) {
	if multiErr := validateMemoRequest(req); multiErr != nil {
		return nil, multiErr
	}

	cus, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         req.CustomerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
			IncludeState:          true,
		},
	})
	if err != nil {
		return nil, err
	}
	if err = s.validateMemoReference(ctx, req, cus); err != nil {
		return nil, err
	}

	control, err := s.billingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	lines, err := s.memoLines(ctx, req)
	if err != nil {
		return nil, err
	}

	return &memoPlan{customer: cus, control: control, lines: lines}, nil
}

func memoQueueItem(
	req *invoiceports.CreateMemoRequest,
	cus *customer.Customer,
	number string,
) *billingqueue.BillingQueueItem {
	return &billingqueue.BillingQueueItem{
		OrganizationID:   req.TenantInfo.OrgID,
		BusinessUnitID:   req.TenantInfo.BuID,
		BillToCustomerID: cus.ID,
		Status:           billingqueue.StatusApproved,
		BillType:         req.BillType,
		Number:           number,
		AdjustmentContext: map[string]any{
			"memo":     true,
			"memoKind": string(memoKindOrDefault(req.MemoKind)),
		},
	}
}

func (s *Service) PreviewMemo(
	ctx context.Context,
	req *invoiceports.CreateMemoRequest,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	plan, err := s.planMemo(ctx, req)
	if err != nil {
		return nil, err
	}

	item := memoQueueItem(req, plan.customer, invoiceassembly.UnnumberedInvoice)
	item.ID = pulid.MustNew("bqi_")
	entity := s.buildMemoEntity(req, item, plan.customer, plan.control, plan.lines)
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	if err = s.posting.RefuseAutoPostingMemo(ctx, req, entity); err != nil {
		return nil, err
	}
	entity.Number = ""

	return entity, nil
}
