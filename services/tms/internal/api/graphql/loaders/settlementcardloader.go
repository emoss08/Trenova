package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type payAdvancesByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req driverpayports.GetPayAdvancesByIDsRequest,
	) ([]*payadvance.PayAdvance, error)
}

type PayAdvanceByIDLoaderFactoryParams struct {
	fx.In

	PayAdvanceRepo driverpayports.PayAdvanceRepository
}

type PayAdvanceByIDLoaderFactory struct {
	advances payAdvancesByIDsGetter
}

func NewPayAdvanceByIDLoaderFactory(
	p PayAdvanceByIDLoaderFactoryParams,
) *PayAdvanceByIDLoaderFactory {
	return &PayAdvanceByIDLoaderFactory{advances: p.PayAdvanceRepo}
}

func newPayAdvanceByIDLoader(f *PayAdvanceByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *payadvance.PayAdvance] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *PayAdvanceByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*payadvance.PayAdvance] {
	return batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*payadvance.PayAdvance, error) {
			return f.advances.GetByIDs(ctx, driverpayports.GetPayAdvancesByIDsRequest{
				TenantInfo: tenantInfo,
				IDs:        ids,
			})
		},
		"Pay advance not found within your organization",
	)
}

type escrowAccountsByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req driverpayports.GetEscrowAccountsByIDsRequest,
	) ([]*payescrow.EscrowAccount, error)
}

type EscrowAccountByIDLoaderFactoryParams struct {
	fx.In

	EscrowAccountRepo driverpayports.EscrowAccountRepository
}

type EscrowAccountByIDLoaderFactory struct {
	accounts escrowAccountsByIDsGetter
}

func NewEscrowAccountByIDLoaderFactory(
	p EscrowAccountByIDLoaderFactoryParams,
) *EscrowAccountByIDLoaderFactory {
	return &EscrowAccountByIDLoaderFactory{accounts: p.EscrowAccountRepo}
}

func newEscrowAccountByIDLoader(f *EscrowAccountByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *payescrow.EscrowAccount] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *EscrowAccountByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*payescrow.EscrowAccount] {
	return batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*payescrow.EscrowAccount, error) {
			return f.accounts.GetByIDs(ctx, driverpayports.GetEscrowAccountsByIDsRequest{
				TenantInfo: tenantInfo,
				IDs:        ids,
			})
		},
		"Escrow account not found within your organization",
	)
}

type settlementDisputesBySettlementIDsLister interface {
	ListBySettlementIDs(
		ctx context.Context,
		req *driversettlementports.ListSettlementDisputesBySettlementIDsRequest,
	) (map[pulid.ID][]*driversettlement.Dispute, error)
}

type SettlementDisputesBySettlementIDLoaderFactoryParams struct {
	fx.In

	SettlementDisputeRepo driversettlementports.SettlementDisputeRepository
}

type SettlementDisputesBySettlementIDLoaderFactory struct {
	disputes settlementDisputesBySettlementIDsLister
}

func NewSettlementDisputesBySettlementIDLoaderFactory(
	p SettlementDisputesBySettlementIDLoaderFactoryParams,
) *SettlementDisputesBySettlementIDLoaderFactory {
	return &SettlementDisputesBySettlementIDLoaderFactory{disputes: p.SettlementDisputeRepo}
}

func newSettlementDisputesBySettlementIDLoader(f *SettlementDisputesBySettlementIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*driversettlement.Dispute] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *SettlementDisputesBySettlementIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*driversettlement.Dispute] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*driversettlement.Dispute, error) {
			return f.disputes.ListBySettlementIDs(
				ctx,
				&driversettlementports.ListSettlementDisputesBySettlementIDsRequest{
					TenantInfo:    tenantInfo,
					SettlementIDs: ids,
				},
			)
		},
	)
}

type carrierInvoiceMatchesByMoveIDsLister interface {
	ListLiveByMoveIDs(
		ctx context.Context,
		req carriersettlementports.ListLiveCarrierInvoiceMatchesByMoveIDsRequest,
	) (map[pulid.ID][]*carriersettlement.InvoiceMatch, error)
}

type CarrierInvoiceMatchesByMoveIDLoaderFactoryParams struct {
	fx.In

	CarrierInvoiceMatchRepo carriersettlementports.CarrierInvoiceMatchRepository
}

type CarrierInvoiceMatchesByMoveIDLoaderFactory struct {
	matches carrierInvoiceMatchesByMoveIDsLister
}

func NewCarrierInvoiceMatchesByMoveIDLoaderFactory(
	p CarrierInvoiceMatchesByMoveIDLoaderFactoryParams,
) *CarrierInvoiceMatchesByMoveIDLoaderFactory {
	return &CarrierInvoiceMatchesByMoveIDLoaderFactory{matches: p.CarrierInvoiceMatchRepo}
}

func newCarrierInvoiceMatchesByMoveIDLoader(f *CarrierInvoiceMatchesByMoveIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*carriersettlement.InvoiceMatch] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *CarrierInvoiceMatchesByMoveIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*carriersettlement.InvoiceMatch] {
	return batchGroupFunc(
		func(
			ctx context.Context,
			ids []pulid.ID,
		) (map[pulid.ID][]*carriersettlement.InvoiceMatch, error) {
			return f.matches.ListLiveByMoveIDs(
				ctx,
				carriersettlementports.ListLiveCarrierInvoiceMatchesByMoveIDsRequest{
					TenantInfo: tenantInfo,
					MoveIDs:    ids,
				},
			)
		},
	)
}
