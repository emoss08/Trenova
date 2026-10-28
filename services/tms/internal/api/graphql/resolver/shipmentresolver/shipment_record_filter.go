package shipmentresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementload"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func (r *QueryResolver) shipmentsInvoicesBill(
	ctx context.Context,
	rawInvoiceIDs []string,
) ([]pulid.ID, error) {
	if rawInvoiceIDs == nil {
		return nil, nil
	}

	authCtx, err := r.RequirePermission(ctx, permission.ResourceInvoice, permission.OpRead)
	if err != nil {
		return nil, err
	}
	invoiceIDs, err := gqlid.ParseIDs(rawInvoiceIDs)
	if err != nil {
		return nil, err
	}
	if len(invoiceIDs) == 0 {
		return []pulid.ID{}, nil
	}

	invoices, err := r.InvoiceService.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		InvoiceIDs: invoiceIDs,
	})
	if err != nil {
		return nil, err
	}

	return invoice.BilledShipmentIDs(invoices...), nil
}

func (r *QueryResolver) shipmentsScope(
	ctx context.Context,
	input *gqlmodel.ShipmentsInput,
) (ids []pulid.ID, scoped bool, err error) {
	sets := make([][]pulid.ID, 0, 3)
	if input.InvoiceIds != nil {
		billed, billErr := r.shipmentsInvoicesBill(ctx, input.InvoiceIds)
		if billErr != nil {
			return nil, true, billErr
		}
		sets = append(sets, billed)
	}
	if input.DriverSettlementIds != nil {
		paid, payErr := r.shipmentsDriverSettlementsPay(ctx, input.DriverSettlementIds)
		if payErr != nil {
			return nil, true, payErr
		}
		sets = append(sets, paid)
	}
	if input.CarrierSettlementIds != nil {
		paid, payErr := r.shipmentsCarrierSettlementsPay(ctx, input.CarrierSettlementIds)
		if payErr != nil {
			return nil, true, payErr
		}
		sets = append(sets, paid)
	}
	if len(sets) == 0 {
		return nil, false, nil
	}
	return sliceutils.Intersect(sets[0], sets[1:]...), true, nil
}

func (r *QueryResolver) shipmentsDriverSettlementsPay(
	ctx context.Context,
	rawSettlementIDs []string,
) ([]pulid.ID, error) {
	authCtx, err := r.RequirePermission(
		ctx,
		permission.ResourceDriverSettlement,
		permission.OpRead,
	)
	if err != nil {
		return nil, err
	}
	settlementIDs, err := gqlid.ParseIDs(rawSettlementIDs)
	if err != nil {
		return nil, err
	}

	lines := make([]settlementload.Line, 0)
	for _, id := range sliceutils.Dedupe(settlementIDs) {
		entity, getErr := r.DriverSettlementRepo.GetByID(
			ctx,
			driversettlementports.GetDriverSettlementByIDRequest{
				ID:           id,
				TenantInfo:   resolvercore.TenantInfo(authCtx),
				IncludeLines: true,
			},
		)
		if getErr != nil {
			return nil, getErr
		}
		lines = append(lines, driversettlement.LoadLines(entity.Lines)...)
	}
	return settlementload.ShipmentIDs(lines), nil
}

func (r *QueryResolver) shipmentsCarrierSettlementsPay(
	ctx context.Context,
	rawSettlementIDs []string,
) ([]pulid.ID, error) {
	authCtx, err := r.RequirePermission(
		ctx,
		permission.ResourceCarrierSettlement,
		permission.OpRead,
	)
	if err != nil {
		return nil, err
	}
	settlementIDs, err := gqlid.ParseIDs(rawSettlementIDs)
	if err != nil {
		return nil, err
	}

	lines := make([]settlementload.Line, 0)
	for _, id := range sliceutils.Dedupe(settlementIDs) {
		entity, getErr := r.CarrierSettlementRepo.GetByID(
			ctx,
			carriersettlementports.GetCarrierSettlementByIDRequest{
				ID:           id,
				TenantInfo:   resolvercore.TenantInfo(authCtx),
				IncludeLines: true,
			},
		)
		if getErr != nil {
			return nil, getErr
		}
		lines = append(lines, carriersettlement.LoadLines(entity.Lines)...)
	}
	return settlementload.ShipmentIDs(lines), nil
}

func emptyShipmentConnection() *gqlmodel.ShipmentConnection {
	total := 0

	return &gqlmodel.ShipmentConnection{
		Edges:           []*gqlmodel.ShipmentEdge{},
		TotalCount:      &total,
		TotalCountExact: true,
		PageInfo:        gqlconnection.PageInfo(false, nil),
	}
}
