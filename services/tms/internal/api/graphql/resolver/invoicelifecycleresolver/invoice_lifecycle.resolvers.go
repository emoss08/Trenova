package invoicelifecycleresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentmapping"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (r *InvoiceResolver) Shipment(ctx context.Context, obj *invoice.Invoice) (*gqlmodel.Shipment, error) {
	if obj == nil || obj.Shipment == nil {
		return nil, nil //nolint:nilnil // an invoice read without its shipment maps to null
	}

	return shipmentmapping.ShipmentToModel(ctx, obj.Shipment)
}

func (r *InvoiceEmailAttemptResolver) Provider(ctx context.Context, obj *invoice.EmailAttempt) (string, error) {
	return string(obj.Provider), nil
}

func (r *InvoiceSendPlanResolver) Headers(ctx context.Context, obj *invoiceports.InvoiceSendPlan) (map[string]any, error) {
	headers := make(map[string]any, len(obj.Headers))
	for key, value := range obj.Headers {
		headers[key] = value
	}

	return headers, nil
}

func (r *MutationResolver) UpdateInvoiceDraft(ctx context.Context, invoiceID string, input gqlmodel.UpdateInvoiceDraftInput) (*invoice.Invoice, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}
	req, err := updateDraftRequest(id, &input, resolvercore.TenantInfo(authCtx))
	if err != nil {
		return nil, err
	}

	return r.InvoiceDraftService.UpdateDraft(ctx, req, actorutil.FromAuthContext(authCtx))
}

func (r *MutationResolver) GenerateInvoicePDF(ctx context.Context, invoiceID string) (*invoiceports.GenerateInvoicePDFResult, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}

	return r.InvoicePDFService.GeneratePDF(
		ctx,
		&invoiceports.InvoicePreviewRequest{
			InvoiceID:  id,
			TenantInfo: resolvercore.TenantInfo(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
}

func (r *MutationResolver) SendInvoice(ctx context.Context, invoiceID string) (*invoiceports.InvoiceSendResult, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpSubmit)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}

	return r.InvoiceSendService.Send(
		ctx,
		&invoiceports.InvoiceSendRequest{
			InvoiceID:  id,
			TenantInfo: resolvercore.TenantInfo(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
}

func (r *MutationResolver) PostInvoice(ctx context.Context, invoiceID string) (*invoice.Invoice, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpPost)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}

	return r.InvoicePostingService.Post(
		ctx,
		&invoiceports.PostInvoiceRequest{
			InvoiceID:   id,
			TenantInfo:  resolvercore.TenantInfo(authCtx),
			TriggeredBy: manualTrigger,
		},
		actorutil.FromAuthContext(authCtx),
	)
}

func (r *QueryResolver) InvoiceEmailAttempts(ctx context.Context, invoiceID string, limit *int, offset *int) (*gqlmodel.InvoiceEmailAttemptPage, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpRead)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}

	tenantInfo := resolvercore.TenantInfo(authCtx)
	page := attemptPage(limit, offset)
	result, err := r.InvoiceQueryService.ListEmailAttempts(
		ctx,
		invoiceports.ListInvoiceEmailAttemptsRequest{
			InvoiceID:  id,
			TenantInfo: tenantInfo,
			Filter: &pagination.QueryOptions{
				TenantInfo: tenantInfo,
				Pagination: page,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return emailAttemptPageToModel(result, page), nil
}

func (r *QueryResolver) InvoiceSendPlan(ctx context.Context, invoiceID string) (*invoiceports.InvoiceSendPlan, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpRead)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}

	return r.InvoiceSendService.PlanSend(ctx, &invoiceports.InvoiceSendPlanRequest{
		InvoiceID:  id,
		TenantInfo: resolvercore.TenantInfo(authCtx),
	})
}

func (r *QueryResolver) InvoicePostPreview(ctx context.Context, invoiceID string) (*invoiceports.InvoicePostPreview, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpPost)
	if err != nil {
		return nil, err
	}

	id, err := gqlid.ParseIDField("invoiceId", invoiceID)
	if err != nil {
		return nil, err
	}

	return r.InvoicePostingService.PreviewPostSummary(
		ctx,
		&invoiceports.PostInvoiceRequest{
			InvoiceID:   id,
			TenantInfo:  resolvercore.TenantInfo(authCtx),
			TriggeredBy: manualTrigger,
		},
		actorutil.FromAuthContext(authCtx),
	)
}
