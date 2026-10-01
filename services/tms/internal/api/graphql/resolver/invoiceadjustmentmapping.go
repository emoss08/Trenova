package resolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func invoiceAdjustmentApprovalsRequestFromGraphQL(
	ctx context.Context,
	input *gqlmodel.InvoiceAdjustmentApprovalsInput,
	tenant pagination.TenantInfo,
) (*repositories.ListApprovalQueueRequest, error) {
	page, err := entityCursorPageFromGraphQL(ctx, gqlCursorPageInput{
		First: input.First,
		After: input.After,
	})
	if err != nil {
		return nil, err
	}

	filters := make([]domaintypes.FieldFilter, 0, 2)
	if input.Kind != nil {
		if !input.Kind.IsValid() {
			return nil, errortypes.NewValidationError(
				"kind",
				errortypes.ErrInvalid,
				"Invalid adjustment kind",
			)
		}
		filters = append(filters, domaintypes.FieldFilter{
			Field:    "kind",
			Operator: dbtype.OpEqual,
			Value:    string(*input.Kind),
		})
	}

	submittedByID, err := optionalPulid(input.SubmittedByID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"submittedById",
			errortypes.ErrInvalid,
			"Invalid submitter",
		)
	}
	if submittedByID != nil {
		filters = append(filters, domaintypes.FieldFilter{
			Field:    "submittedById",
			Operator: dbtype.OpEqual,
			Value:    submittedByID.String(),
		})
	}

	query := ""
	if input.Query != nil {
		query = strings.TrimSpace(*input.Query)
	}

	return &repositories.ListApprovalQueueRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo:   tenant,
			Pagination:   pagination.Info{Limit: page.Cursor.Limit},
			Query:        query,
			FieldFilters: filters,
		},
		Cursor: page.Cursor,
	}, nil
}

func invoiceAdjustmentApprovalConnectionToModel(
	result *pagination.CursorListResult[*repositories.InvoiceAdjustmentApprovalQueueItem],
) (*gqlmodel.InvoiceAdjustmentApprovalQueueConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(
			node *repositories.InvoiceAdjustmentApprovalQueueItem,
			cursor string,
		) *gqlmodel.InvoiceAdjustmentApprovalQueueEdge {
			return &gqlmodel.InvoiceAdjustmentApprovalQueueEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.InvoiceAdjustmentApprovalQueueEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.InvoiceAdjustmentApprovalQueueConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func parseInvoiceAdjustmentID(raw string) (pulid.ID, error) {
	id, err := pulid.MustParse(raw)
	if err != nil {
		return pulid.Nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalid,
			"Invalid invoice adjustment",
		)
	}

	return id, nil
}
