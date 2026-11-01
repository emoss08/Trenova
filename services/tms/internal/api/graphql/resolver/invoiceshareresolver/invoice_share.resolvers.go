package invoiceshareresolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceshare"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (r *MutationResolver) ShareInvoice(ctx context.Context, invoiceID string, input gqlmodel.ShareInvoiceInput) (*invoiceports.ShareInvoiceResult, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpRead)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}
	userIDs, err := gqlid.ParseIDListField("userIds", input.UserIds)
	if err != nil {
		return nil, err
	}

	return r.InvoiceShareService.Share(
		ctx,
		&invoiceports.ShareInvoiceRequest{
			TenantInfo: resolvercore.TenantInfo(authCtx),
			InvoiceID:  id,
			UserIDs:    userIDs,
			Note:       strings.TrimSpace(gqlscalar.StringValue(input.Note)),
			Tab:        input.Tab,
		},
		actorutil.FromAuthContext(authCtx),
	)
}

func (r *QueryResolver) InvoiceShares(ctx context.Context, invoiceID string) ([]*invoiceshare.InvoiceShare, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpRead)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}

	shares, err := r.InvoiceShareService.List(ctx, &invoiceports.ListInvoiceSharesRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		InvoiceID:  id,
	})
	if err != nil {
		return nil, err
	}
	if shares == nil {
		return []*invoiceshare.InvoiceShare{}, nil
	}

	return shares, nil
}

func (r *QueryResolver) InvoiceShareCandidates(ctx context.Context, invoiceID string, query *string, limit *int) ([]*invoiceports.ShareCandidate, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpRead)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}

	result, err := r.InvoiceShareService.ListCandidates(
		ctx,
		&invoiceports.ListShareCandidatesRequest{
			TenantInfo: resolvercore.TenantInfo(authCtx),
			InvoiceID:  id,
			Query:      strings.TrimSpace(gqlscalar.StringValue(query)),
			Pagination: pagination.Info{Limit: candidateLimit(limit)},
		},
	)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Items == nil {
		return []*invoiceports.ShareCandidate{}, nil
	}

	return result.Items, nil
}
