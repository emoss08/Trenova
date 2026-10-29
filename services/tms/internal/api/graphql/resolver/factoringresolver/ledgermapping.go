package factoringresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
)

func fundingRequestFromModel(
	in *gqlmodel.RecordFactoringFundingInput,
	authCtx *authctx.AuthContext,
) (*factoringledgerports.RecordFundingRequest, error) {
	submissionID, err := parseID(in.SubmissionID, "submissionId")
	if err != nil {
		return nil, err
	}
	items := make([]factoringledgerports.FundingItemInput, 0, len(in.Items))
	for _, item := range in.Items {
		if item == nil {
			continue
		}
		itemID, parseErr := parseID(item.ItemID, "items.itemId")
		if parseErr != nil {
			return nil, parseErr
		}
		items = append(items, factoringledgerports.FundingItemInput{
			ItemID:       itemID,
			Outcome:      item.Outcome,
			AdvanceMinor: int64Ptr(item.AdvanceMinor),
			FeeMinor:     int64Ptr(item.FeeMinor),
			Reason:       stringutils.FromPtr(item.Reason),
		})
	}
	return &factoringledgerports.RecordFundingRequest{
		TenantInfo:   resolvercore.TenantInfo(authCtx),
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
) (*factoringledgerports.ReserveMovementRequest, error) {
	submissionID, err := parseID(in.SubmissionID, "submissionId")
	if err != nil {
		return nil, err
	}
	itemID, err := parseID(in.ItemID, "itemId")
	if err != nil {
		return nil, err
	}
	return &factoringledgerports.ReserveMovementRequest{
		TenantInfo:   resolvercore.TenantInfo(authCtx),
		SubmissionID: submissionID,
		ItemID:       itemID,
		Version:      int64(in.Version),
		OccurredOn:   int64(in.OccurredOn),
		CashMinor:    int64Ptr(in.CashMinor),
		Reason:       stringutils.FromPtr(in.Reason),
	}, nil
}

func reserveConnectionToModel(
	result *pagination.CursorListResult[*factoringledger.ReserveEntry],
) (*gqlmodel.FactoringReserveEntryConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *factoringledger.ReserveEntry, cursor string) *gqlmodel.FactoringReserveEntryEdge {
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

func invoiceEffectToModel(effect factoringledger.InvoiceEffect) gqlmodel.FactoringInvoiceEffect {
	switch effect {
	case factoringledger.InvoiceEffectSettle:
		return gqlmodel.FactoringInvoiceEffectSettle
	case factoringledger.InvoiceEffectReopen:
		return gqlmodel.FactoringInvoiceEffectReopen
	case factoringledger.InvoiceEffectNone:
		return gqlmodel.FactoringInvoiceEffectNone
	default:
		return gqlmodel.FactoringInvoiceEffectNone
	}
}
