package invoiceadjustmentresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultQueueLimit = 100
	maxQueueLimit     = 100
)

func queueQueryOptions(
	input *gqlmodel.InvoiceAdjustmentQueueInput,
	tenantInfo pagination.TenantInfo,
) pagination.QueryOptions {
	opts := pagination.QueryOptions{
		TenantInfo: tenantInfo,
		Pagination: pagination.Info{Limit: defaultQueueLimit},
		Query:      strings.TrimSpace(gqlscalar.StringValue(input.Query)),
	}
	if input.Limit != nil && *input.Limit > 0 {
		opts.Pagination.Limit = min(*input.Limit, maxQueueLimit)
	}
	if input.Offset != nil && *input.Offset > 0 {
		opts.Pagination.Offset = *input.Offset
	}
	if status := strings.TrimSpace(gqlscalar.StringValue(input.Status)); status != "" {
		opts.FieldFilters = []domaintypes.FieldFilter{{
			Field:    "status",
			Operator: dbtype.OpEqual,
			Value:    status,
		}}
	}

	return opts
}

func batchPageToModel(
	result *pagination.ListResult[*invoiceadjustment.InvoiceAdjustmentBatch],
) *gqlmodel.InvoiceAdjustmentBatchPage {
	page := &gqlmodel.InvoiceAdjustmentBatchPage{
		Items: []*invoiceadjustment.InvoiceAdjustmentBatch{},
	}
	if result == nil {
		return page
	}
	if result.Items != nil {
		page.Items = result.Items
	}
	page.Total = result.Total
	page.TotalExact = result.TotalExact

	return page
}

func reconciliationPageToModel(
	result *pagination.ListResult[*invoiceadjustmentports.InvoiceAdjustmentReconciliationQueueItem],
) *gqlmodel.InvoiceAdjustmentReconciliationPage {
	page := &gqlmodel.InvoiceAdjustmentReconciliationPage{
		Items: []*invoiceadjustmentports.InvoiceAdjustmentReconciliationQueueItem{},
	}
	if result == nil {
		return page
	}
	if result.Items != nil {
		page.Items = result.Items
	}
	page.Total = result.Total
	page.TotalExact = result.TotalExact

	return page
}

func previewErrorsToModel(errors map[string][]string) map[string]any {
	out := make(map[string]any, len(errors))
	for field, messages := range errors {
		values := make([]any, 0, len(messages))
		for _, message := range messages {
			values = append(values, message)
		}
		out[field] = values
	}

	return out
}

func updateDraftRequestFromInput(
	adjustmentID pulid.ID,
	input *gqlmodel.InvoiceAdjustmentDraftInput,
	tenantInfo pagination.TenantInfo,
) (*invoiceadjustmentports.UpdateDraftInvoiceAdjustmentRequest, error) {
	documentIDs, err := gqlid.ParseIDListField("referencedDocumentIds", input.ReferencedDocumentIds)
	if err != nil {
		return nil, err
	}
	lines, err := adjustmentLinesFromInput(input.Lines)
	if err != nil {
		return nil, err
	}

	req := &invoiceadjustmentports.UpdateDraftInvoiceAdjustmentRequest{
		AdjustmentID:          adjustmentID,
		Kind:                  input.Kind,
		RebillStrategy:        invoiceadjustment.RebillStrategyCloneExact,
		Reason:                gqlscalar.StringValue(input.Reason),
		ReferencedDocumentIDs: documentIDs,
		Lines:                 lines,
		TenantInfo:            tenantInfo,
	}
	if input.RebillStrategy != nil {
		req.RebillStrategy = *input.RebillStrategy
	}

	return req, nil
}

func adjustmentLinesFromInput(
	inputs []*gqlmodel.InvoiceAdjustmentLineInput,
) ([]*invoiceadjustmentports.InvoiceAdjustmentLineInput, error) {
	lines := make([]*invoiceadjustmentports.InvoiceAdjustmentLineInput, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		line, err := adjustmentLineFromInput(input)
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}

	return lines, nil
}

func adjustmentLineFromInput(
	input *gqlmodel.InvoiceAdjustmentLineInput,
) (*invoiceadjustmentports.InvoiceAdjustmentLineInput, error) {
	lineID, err := gqlid.ParseIDField("lines.originalLineId", input.OriginalLineID)
	if err != nil {
		return nil, err
	}
	creditQuantity, err := gqldecimal.ParseDecimalField(
		"lines.creditQuantity",
		input.CreditQuantity,
		true,
	)
	if err != nil {
		return nil, err
	}
	creditAmount, err := gqldecimal.ParseDecimalField(
		"lines.creditAmount",
		input.CreditAmount,
		true,
	)
	if err != nil {
		return nil, err
	}
	rebillQuantity, err := gqldecimal.ParseDecimalField(
		"lines.rebillQuantity",
		input.RebillQuantity,
		true,
	)
	if err != nil {
		return nil, err
	}
	rebillAmount, err := gqldecimal.ParseDecimalField(
		"lines.rebillAmount",
		input.RebillAmount,
		true,
	)
	if err != nil {
		return nil, err
	}
	payload := input.ReplacementPayload
	if payload == nil {
		payload = map[string]any{}
	}

	return &invoiceadjustmentports.InvoiceAdjustmentLineInput{
		OriginalLineID:     lineID,
		CreditQuantity:     creditQuantity,
		CreditAmount:       creditAmount,
		RebillQuantity:     rebillQuantity,
		RebillAmount:       rebillAmount,
		Description:        gqlscalar.StringValue(input.Description),
		ReplacementPayload: payload,
	}, nil
}
