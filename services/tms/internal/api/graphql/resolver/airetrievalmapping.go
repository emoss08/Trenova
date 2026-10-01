package resolver

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

func omittableValue[T any](
	multiErr *errortypes.MultiError,
	field, label string,
	value graphql.Omittable[*T],
) *T {
	if !value.IsSet() {
		return nil
	}

	ptr := value.Value()
	if ptr == nil {
		multiErr.Add(field, errortypes.ErrRequired, "{0} cannot be cleared", label)
		return nil
	}

	out := *ptr
	return &out
}

func aiRetrievalSettingsPatchFromInput(
	tenant pagination.TenantInfo,
	input *gqlmodel.AIRetrievalSettingsPatchInput,
) (*services.UpdateAIRetrievalSettingsRequest, error) {
	multiErr := errortypes.NewMultiError()
	req := &services.UpdateAIRetrievalSettingsRequest{
		TenantInfo: tenant,
		MemoryEnabled: omittableValue(
			multiErr, "memoryEnabled", "Memory indexing", input.MemoryEnabled,
		),
		DocumentsEnabled: omittableValue(
			multiErr, "documentsEnabled", "Document indexing", input.DocumentsEnabled,
		),
		InboundMessagesEnabled: omittableValue(
			multiErr, "inboundMessagesEnabled", "Inbound email indexing",
			input.InboundMessagesEnabled,
		),
		Paused: omittableValue(multiErr, "paused", "Paused", input.Paused),
	}

	if raw := omittableValue(
		multiErr, "monthlyIndexingBudgetUsd", "Monthly indexing budget",
		input.MonthlyIndexingBudgetUsd,
	); raw != nil {
		budget, err := decimal.NewFromString(*raw)
		if err != nil {
			multiErr.Add(
				"monthlyIndexingBudgetUsd",
				errortypes.ErrInvalid,
				"Must be a valid decimal number",
			)
		} else {
			req.MonthlyIndexingBudgetUSD = &budget
		}
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return req, nil
}

func aiRetrievalFailedEntriesToModel(
	page *services.AIRetrievalFailedEntryPage,
) *gqlmodel.AIRetrievalFailedEntryConnection {
	out := &gqlmodel.AIRetrievalFailedEntryConnection{
		Edges:      make([]*gqlmodel.AIRetrievalFailedEntryEdge, 0, len(page.Edges)),
		TotalCount: page.TotalCount,
	}
	last := ""
	for _, edge := range page.Edges {
		out.Edges = append(out.Edges, &gqlmodel.AIRetrievalFailedEntryEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		})
		last = edge.Cursor
	}
	out.PageInfo = pageInfoFor(page.HasNextPage, last)

	return out
}
