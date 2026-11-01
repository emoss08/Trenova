package invoicestatementresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *CommitInvoiceRunResultResolver) RunID(ctx context.Context, obj *invoicerunports.CommitInvoiceRunResult) (string, error) {
	if obj == nil || obj.Run == nil {
		return "", nil
	}

	return obj.Run.ID.String(), nil
}

func (r *CommitInvoiceRunResultResolver) RunNumber(ctx context.Context, obj *invoicerunports.CommitInvoiceRunResult) (string, error) {
	if obj == nil || obj.Run == nil {
		return "", nil
	}

	return obj.Run.Number, nil
}

func (r *MutationResolver) BillStatementNow(ctx context.Context, customerID string, input gqlmodel.BillStatementNowInput) (*invoicerunports.CommitInvoiceRunResult, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoiceRun, permission.OpApprove)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("customerId", customerID)
	if err != nil {
		return nil, err
	}
	exclusions, err := statementExclusions(input.Exclude)
	if err != nil {
		return nil, err
	}

	return r.InvoiceStatementService.BillStatementNow(
		ctx,
		&invoicerunports.BillStatementNowRequest{
			TenantInfo: resolvercore.TenantInfo(authCtx),
			CustomerID: id,
			Reason:     gqlscalar.StringValue(input.Reason),
			Exclude:    exclusions,
		},
		actorutil.FromAuthContext(authCtx),
	)
}

func (r *OpenStatementResolver) TotalAmount(ctx context.Context, obj *invoicerunports.OpenStatement) (string, error) {
	return obj.TotalAmount.String(), nil
}

func (r *OpenStatementResolver) MinimumAmount(ctx context.Context, obj *invoicerunports.OpenStatement) (*string, error) {
	return gqldecimal.NullDecimalStringPtr(obj.MinimumAmount), nil
}

func (r *OpenStatementResolver) HeldAmount(ctx context.Context, obj *invoicerunports.OpenStatement) (string, error) {
	return obj.HeldAmount.String(), nil
}

func (r *QueryResolver) OpenStatements(ctx context.Context) ([]*invoicerunports.OpenStatement, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoiceRun, permission.OpRead)
	if err != nil {
		return nil, err
	}

	statements, err := r.InvoiceStatementService.ListOpenStatements(
		ctx,
		&invoicerunports.ListOpenStatementsRequest{TenantInfo: resolvercore.TenantInfo(authCtx)},
	)
	if err != nil {
		return nil, err
	}
	if statements == nil {
		return []*invoicerunports.OpenStatement{}, nil
	}

	return statements, nil
}

func (r *QueryResolver) OpenStatement(ctx context.Context, customerID string) (*invoicerunports.OpenStatement, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoiceRun, permission.OpRead)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("customerId", customerID)
	if err != nil {
		return nil, err
	}

	return r.InvoiceStatementService.GetOpenStatement(
		ctx,
		&invoicerunports.ListOpenStatementsRequest{
			TenantInfo: resolvercore.TenantInfo(authCtx),
			CustomerID: id,
		},
	)
}

func (r *StatementGroupResolver) TotalAmount(ctx context.Context, obj *invoicerunports.StatementGroup) (string, error) {
	return obj.TotalAmount.String(), nil
}

func (r *StatementShipmentResolver) Amount(ctx context.Context, obj *invoicerunports.StatementShipment) (string, error) {
	return obj.Amount.String(), nil
}

func statementExclusions(
	inputs []*gqlmodel.StatementExclusionInput,
) ([]invoicerunports.StatementExclusion, error) {
	exclusions := make([]invoicerunports.StatementExclusion, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		itemID, err := gqlid.ParseIDField("exclude.billingQueueItemId", input.BillingQueueItemID)
		if err != nil {
			return nil, err
		}
		exclusions = append(exclusions, invoicerunports.StatementExclusion{
			BillingQueueItemID: itemID,
			Reason:             input.Reason,
		})
	}

	return exclusions, nil
}
