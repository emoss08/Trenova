package factoringresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
)

func fundingRequestFromModel(
	in *gqlmodel.RecordFactoringFundingInput,
	authCtx *authctx.AuthContext,
) (*factoringports.RecordFundingRequest, error) {
	submissionID, err := parseID(in.SubmissionID, "submissionId")
	if err != nil {
		return nil, err
	}
	items := make([]factoringports.FundingItemInput, 0, len(in.Items))
	for _, item := range in.Items {
		if item == nil {
			continue
		}
		itemID, parseErr := parseID(item.ItemID, "items.itemId")
		if parseErr != nil {
			return nil, parseErr
		}
		items = append(items, factoringports.FundingItemInput{
			ItemID:       itemID,
			Outcome:      item.Outcome,
			AdvanceMinor: int64Ptr(item.AdvanceMinor),
			FeeMinor:     int64Ptr(item.FeeMinor),
			Reason:       stringutils.FromPtr(item.Reason),
		})
	}
	return &factoringports.RecordFundingRequest{
		TenantInfo:   base.TenantInfo(authCtx),
		SubmissionID: submissionID,
		Version:      int64(in.Version),
		FundedOn:     int64(in.FundedOn),
		Items:        items,
		Memo:         stringutils.FromPtr(in.Memo),
	}, nil
}

func movementRequestFromModel(
	in *gqlmodel.FactoringReserveMovementInput,
	authCtx *authctx.AuthContext,
) (*factoringports.ReserveMovementRequest, error) {
	submissionID, err := parseID(in.SubmissionID, "submissionId")
	if err != nil {
		return nil, err
	}
	itemID, err := parseID(in.ItemID, "itemId")
	if err != nil {
		return nil, err
	}
	return &factoringports.ReserveMovementRequest{
		TenantInfo:   base.TenantInfo(authCtx),
		SubmissionID: submissionID,
		ItemID:       itemID,
		Version:      int64(in.Version),
		OccurredOn:   int64(in.OccurredOn),
		CashMinor:    int64Ptr(in.CashMinor),
		Reason:       stringutils.FromPtr(in.Reason),
	}, nil
}

func reserveConnectionToModel(
	result *pagination.CursorListResult[*factoring.ReserveEntry],
) (*gqlmodel.FactoringReserveEntryConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *factoring.ReserveEntry, cursor string) *gqlmodel.FactoringReserveEntryEdge {
			return &gqlmodel.FactoringReserveEntryEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.FactoringReserveEntryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.FactoringReserveEntryConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func invoiceEffectToModel(effect factoring.InvoiceEffect) gqlmodel.FactoringInvoiceEffect {
	switch effect {
	case factoring.InvoiceEffectSettle:
		return gqlmodel.FactoringInvoiceEffectSettle
	case factoring.InvoiceEffectReopen:
		return gqlmodel.FactoringInvoiceEffectReopen
	case factoring.InvoiceEffectNone:
		return gqlmodel.FactoringInvoiceEffectNone
	default:
		return gqlmodel.FactoringInvoiceEffectNone
	}
}
