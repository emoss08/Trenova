package invoicelifecycleresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	manualTrigger           = "manual"
	defaultAttemptPageLimit = 20
	maxAttemptPageLimit     = 100
)

func attemptPage(limit, offset *int) pagination.Info {
	page := pagination.Info{Limit: defaultAttemptPageLimit}
	if limit != nil && *limit > 0 {
		page.Limit = min(*limit, maxAttemptPageLimit)
	}
	if offset != nil && *offset > 0 {
		page.Offset = *offset
	}

	return page
}

func emailAttemptPageToModel(
	result *pagination.ListResult[*invoice.EmailAttempt],
	page pagination.Info,
) *gqlmodel.InvoiceEmailAttemptPage {
	out := &gqlmodel.InvoiceEmailAttemptPage{Items: []*invoice.EmailAttempt{}}
	if result == nil {
		return out
	}
	if result.Items != nil {
		out.Items = result.Items
	}
	out.Total = result.Total
	out.TotalExact = result.TotalExact
	out.HasMore = pagination.HasMore(result, page)

	return out
}

func updateDraftRequest(
	invoiceID pulid.ID,
	input *gqlmodel.UpdateInvoiceDraftInput,
	tenantInfo pagination.TenantInfo,
) (*invoiceports.UpdateInvoiceDraftRequest, error) {
	req := &invoiceports.UpdateInvoiceDraftRequest{
		InvoiceID:              invoiceID,
		TenantInfo:             tenantInfo,
		Memo:                   input.Memo,
		RemittanceInstructions: input.RemittanceInstructions,
		EmailSubject:           input.EmailSubject,
		EmailBody:              input.EmailBody,
		EmailTo:                optionalList(input.EmailTo),
		EmailCC:                optionalList(input.EmailCc),
		EmailBCC:               optionalList(input.EmailBCC),
	}
	if input.AttachmentIds != nil {
		ids, err := gqlid.ParseIDListField("attachmentIds", input.AttachmentIds)
		if err != nil {
			return nil, err
		}
		req.AttachmentIDs = &ids
	}

	return req, nil
}

func optionalList(values []string) *[]string {
	if values == nil {
		return nil
	}

	return &values
}
