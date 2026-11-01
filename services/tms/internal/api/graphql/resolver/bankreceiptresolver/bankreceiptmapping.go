package bankreceiptresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptbatchports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func importBatchRequestFromInput(
	input *gqlmodel.ImportBankReceiptBatchInput,
) *bankreceiptbatchports.ImportBankReceiptBatchRequest {
	receipts := make([]*bankreceiptbatchports.ImportBankReceiptBatchLine, 0, len(input.Receipts))
	for _, line := range input.Receipts {
		receipts = append(receipts, &bankreceiptbatchports.ImportBankReceiptBatchLine{
			ReceiptDate:     int64(line.ReceiptDate),
			AmountMinor:     int64(line.AmountMinor),
			ReferenceNumber: line.ReferenceNumber,
			Memo:            gqlscalar.StringValue(line.Memo),
		})
	}

	return &bankreceiptbatchports.ImportBankReceiptBatchRequest{
		Source:    input.Source,
		Reference: input.Reference,
		Receipts:  receipts,
	}
}

func getWorkItemRequest(
	id string,
	authCtx *authctx.AuthContext,
) (*bankreceiptworkitemports.GetBankReceiptWorkItemRequest, error) {
	workItemID, err := pulid.MustParse(id)
	if err != nil {
		return nil, err
	}

	return &bankreceiptworkitemports.GetBankReceiptWorkItemRequest{
		WorkItemID: workItemID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
	}, nil
}
