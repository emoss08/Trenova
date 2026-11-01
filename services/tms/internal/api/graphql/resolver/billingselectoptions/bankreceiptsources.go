package billingselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptbatchports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

type Sources struct {
	*resolvercore.Core
	BankReceiptBatchService bankreceiptbatchports.BankReceiptBatchService
}

func (r *Sources) BankReceiptSource(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if _, err := r.RequirePermission(
		ctx,
		permission.ResourceBankReceipt,
		permission.OpRead,
	); err != nil {
		return nil, err
	}

	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			items = append(items, bankReceiptSourceItem(id.String(), id.String()))
		}
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.BankReceiptBatchService.DistinctSources(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		func(option *bankreceiptbatchports.BankReceiptBatchSourceOption) selectoption.Item {
			return bankReceiptSourceItem(option.Value, option.Label)
		},
	)
}

func bankReceiptSourceItem(value, label string) selectoption.Item {
	return selectoption.ItemFor(
		&gqlmodel.SelectOption{ID: value, Label: label},
		0,
		pulid.ID(value),
	)
}
