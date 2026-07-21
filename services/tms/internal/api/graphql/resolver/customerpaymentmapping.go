package resolver

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

func customerPaymentConnectionToModel(
	result *pagination.CursorListResult[*customerpayment.Payment],
) (*gqlmodel.CustomerPaymentConnection, error) {
	page, err := entityCursorConnection(
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
) ([]*serviceports.CustomerPaymentApplicationInput, error) {
	applications := make([]*serviceports.CustomerPaymentApplicationInput, 0, len(inputs))
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
		application := &serviceports.CustomerPaymentApplicationInput{
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
) (*serviceports.PostCustomerPaymentRequest, error) {
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

	currencyCode := strings.TrimSpace(stringValue(input.CurrencyCode))
	if currencyCode == "" {
		currencyCode = money.DefaultCurrencyCode
	}

	return &serviceports.PostCustomerPaymentRequest{
		CustomerID:      customerID,
		PaymentDate:     int64(input.PaymentDate),
		AccountingDate:  int64(input.AccountingDate),
		AmountMinor:     int64(input.AmountMinor),
		PaymentMethod:   input.PaymentMethod,
		ReferenceNumber: stringValue(input.ReferenceNumber),
		Memo:            stringValue(input.Memo),
		CurrencyCode:    currencyCode,
		Applications:    applications,
		TenantInfo:      tenantInfo,
	}, nil
}

func applyCustomerPaymentRequestFromInput(
	input *gqlmodel.ApplyCustomerPaymentInput,
	tenantInfo pagination.TenantInfo,
) (*serviceports.ApplyCustomerPaymentRequest, error) {
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

	return &serviceports.ApplyCustomerPaymentRequest{
		PaymentID:      paymentID,
		AccountingDate: int64(input.AccountingDate),
		Applications:   applications,
		TenantInfo:     tenantInfo,
	}, nil
}

func reverseCustomerPaymentRequestFromInput(
	input *gqlmodel.ReverseCustomerPaymentInput,
	tenantInfo pagination.TenantInfo,
) (*serviceports.ReverseCustomerPaymentRequest, error) {
	paymentID, err := pulid.MustParse(input.PaymentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"paymentId",
			errortypes.ErrInvalid,
			"Invalid payment",
		)
	}

	return &serviceports.ReverseCustomerPaymentRequest{
		PaymentID:      paymentID,
		AccountingDate: int64(input.AccountingDate),
		Reason:         stringValue(input.Reason),
		TenantInfo:     tenantInfo,
	}, nil
}
