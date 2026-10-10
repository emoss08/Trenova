package customerpaymentresolver

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

func customerPaymentConnectionToModel(
	result *pagination.CursorListResult[*customerpayment.Payment],
) (*gqlmodel.CustomerPaymentConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *customerpayment.Payment, cursor string) *gqlmodel.CustomerPaymentEdge {
			return &gqlmodel.CustomerPaymentEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.CustomerPaymentEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.CustomerPaymentConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func customerPaymentApplicationsFromInput(
	inputs []*gqlmodel.CustomerPaymentApplicationInput,
) ([]*customerpaymentports.CustomerPaymentApplicationInput, error) {
	applications := make([]*customerpaymentports.CustomerPaymentApplicationInput, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		invoiceID, err := pulid.MustParse(input.InvoiceID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("applications[%d].invoiceId", idx),
				errortypes.ErrInvalid,
				"Invalid invoice",
			)
		}
		application := &customerpaymentports.CustomerPaymentApplicationInput{
			InvoiceID:          invoiceID,
			AppliedAmountMinor: int64(input.AppliedAmountMinor),
		}
		if input.ShortPayAmountMinor != nil {
			application.ShortPayAmountMinor = int64(*input.ShortPayAmountMinor)
		}
		applications = append(applications, application)
	}
	return applications, nil
}

func postCustomerPaymentRequestFromInput(
	input *gqlmodel.PostCustomerPaymentInput,
	tenantInfo pagination.TenantInfo,
) (*customerpaymentports.PostCustomerPaymentRequest, error) {
	customerID, err := pulid.MustParse(input.CustomerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"customerId",
			errortypes.ErrInvalid,
			"Invalid customer",
		)
	}

	applications, err := customerPaymentApplicationsFromInput(input.Applications)
	if err != nil {
		return nil, err
	}

	currencyCode := strings.TrimSpace(base.StringValue(input.CurrencyCode))
	if currencyCode == "" {
		currencyCode = money.DefaultCurrencyCode
	}

	return &customerpaymentports.PostCustomerPaymentRequest{
		CustomerID:      customerID,
		PaymentDate:     int64(input.PaymentDate),
		AccountingDate:  int64(input.AccountingDate),
		AmountMinor:     int64(input.AmountMinor),
		PaymentMethod:   input.PaymentMethod,
		ReferenceNumber: base.StringValue(input.ReferenceNumber),
		Memo:            base.StringValue(input.Memo),
		CurrencyCode:    currencyCode,
		Applications:    applications,
		TenantInfo:      tenantInfo,
	}, nil
}

func applyCustomerPaymentRequestFromInput(
	input *gqlmodel.ApplyCustomerPaymentInput,
	tenantInfo pagination.TenantInfo,
) (*customerpaymentports.ApplyCustomerPaymentRequest, error) {
	paymentID, err := pulid.MustParse(input.PaymentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"paymentId",
			errortypes.ErrInvalid,
			"Invalid payment",
		)
	}

	applications, err := customerPaymentApplicationsFromInput(input.Applications)
	if err != nil {
		return nil, err
	}

	return &customerpaymentports.ApplyCustomerPaymentRequest{
		PaymentID:      paymentID,
		AccountingDate: int64(input.AccountingDate),
		Applications:   applications,
		TenantInfo:     tenantInfo,
	}, nil
}

func reverseCustomerPaymentRequestFromInput(
	input *gqlmodel.ReverseCustomerPaymentInput,
	tenantInfo pagination.TenantInfo,
) (*customerpaymentports.ReverseCustomerPaymentRequest, error) {
	paymentID, err := pulid.MustParse(input.PaymentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"paymentId",
			errortypes.ErrInvalid,
			"Invalid payment",
		)
	}

	return &customerpaymentports.ReverseCustomerPaymentRequest{
		PaymentID:      paymentID,
		AccountingDate: int64(input.AccountingDate),
		Reason:         base.StringValue(input.Reason),
		TenantInfo:     tenantInfo,
	}, nil
}
