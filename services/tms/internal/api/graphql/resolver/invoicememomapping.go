package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const secondsPerDay = 86400

func createMemoRequestFromInput(
	input *gqlmodel.CreateMemoInput,
	tenantInfo pagination.TenantInfo,
) (*serviceports.CreateMemoRequest, error) {
	customerID, err := pulid.MustParse(input.CustomerID)
	if err != nil {
		return nil, errortypes.NewValidationError("customerId", errortypes.ErrInvalid, "Invalid customer")
	}

	var referenceInvoiceID pulid.ID
	if input.ReferenceInvoiceID != nil && *input.ReferenceInvoiceID != "" {
		referenceInvoiceID, err = pulid.MustParse(*input.ReferenceInvoiceID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"referenceInvoiceId",
				errortypes.ErrInvalid,
				"Invalid reference invoice",
			)
		}
	}

	lines, err := memoLinesFromInput(input.Lines)
	if err != nil {
		return nil, err
	}

	req := &serviceports.CreateMemoRequest{
		TenantInfo:         tenantInfo,
		CustomerID:         customerID,
		BillType:           input.BillType,
		Lines:              lines,
		ReferenceInvoiceID: referenceInvoiceID,
		Reason:             input.Reason,
		Memo:               stringutils.FromPtr(input.Memo),
	}
	if input.InvoiceDate != nil {
		req.InvoiceDate = int64(*input.InvoiceDate)
	}
	if input.MemoKind != nil {
		req.MemoKind = *input.MemoKind
	}
	if input.AutoPost != nil {
		req.AutoPost = *input.AutoPost
	}

	return req, nil
}

func memoLinesFromInput(inputs []*gqlmodel.MemoLineInput) ([]*serviceports.CreateMemoLineInput, error) {
	lines := make([]*serviceports.CreateMemoLineInput, 0, len(inputs))
	multiErr := errortypes.NewMultiError()
	for idx, in := range inputs {
		if in == nil {
			multiErr.WithIndex("lines", idx).Add("description", errortypes.ErrRequired, "Line is required")
			continue
		}
		line := &serviceports.CreateMemoLineInput{Description: in.Description}
		amount, err := decimal.NewFromString(in.Amount)
		if err != nil {
			multiErr.WithIndex("lines", idx).Add("amount", errortypes.ErrInvalid, "Must be a valid decimal number")
		}
		line.Amount = amount
		if in.Quantity != nil && *in.Quantity != "" {
			quantity, qErr := decimal.NewFromString(*in.Quantity)
			if qErr != nil {
				multiErr.WithIndex("lines", idx).Add("quantity", errortypes.ErrInvalid, "Must be a valid decimal number")
			}
			line.Quantity = quantity
		}
		if in.AccessorialChargeID != nil && *in.AccessorialChargeID != "" {
			accessorialID, aErr := pulid.MustParse(*in.AccessorialChargeID)
			if aErr != nil {
				multiErr.WithIndex("lines", idx).Add("accessorialChargeId", errortypes.ErrInvalid, "Invalid accessorial charge")
			}
			line.AccessorialChargeID = accessorialID
		}
		lines = append(lines, line)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return lines, nil
}

// invoiceDaysPastDue is how many whole days an open invoice is past its due
// date at the given instant, or nil when nothing is owed or it is not yet due.
func invoiceDaysPastDue(entity *invoice.Invoice, now int64) *int {
	if entity == nil || !entity.IsOpen() || entity.DueDate == nil {
		return nil
	}
	overdue := now - *entity.DueDate
	if overdue < secondsPerDay {
		return nil
	}
	days := int(overdue / secondsPerDay)

	return &days
}
