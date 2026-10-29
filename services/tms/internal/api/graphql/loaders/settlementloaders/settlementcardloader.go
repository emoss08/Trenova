package settlementloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *PayAdvanceByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*payadvance.PayAdvance] {
	return loadercore.BatchByIDFunc(
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *EscrowAccountByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*payescrow.EscrowAccount] {
	return loadercore.BatchByIDFunc(
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *SettlementDisputesBySettlementIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*driversettlement.Dispute] {
	return loadercore.BatchGroupFunc(
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *CarrierInvoiceMatchesByMoveIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*carriersettlement.InvoiceMatch] {
	return loadercore.BatchGroupFunc(
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

var (
	payAdvanceByIDKey                   = loadercore.NewKey(newPayAdvanceByIDLoader)
	escrowAccountByIDKey                = loadercore.NewKey(newEscrowAccountByIDLoader)
	settlementDisputesBySettlementIDKey = loadercore.NewKey(
		newSettlementDisputesBySettlementIDLoader,
	)
	carrierInvoiceMatchesByMoveIDKey = loadercore.NewKey(newCarrierInvoiceMatchesByMoveIDLoader)
)

func PayAdvanceByID(ctx context.Context, key string) (*payadvance.PayAdvance, error) {
	return loadercore.Load(ctx, payAdvanceByIDKey, key)
}

func (f *PayAdvanceByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func EscrowAccountByID(ctx context.Context, key string) (*payescrow.EscrowAccount, error) {
	return loadercore.Load(ctx, escrowAccountByIDKey, key)
}

func (f *EscrowAccountByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func SettlementDisputesBySettlementID(
	ctx context.Context,
	key string,
) ([]*driversettlement.Dispute, error) {
	return loadercore.Load(ctx, settlementDisputesBySettlementIDKey, key)
}

func (f *SettlementDisputesBySettlementIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func CarrierInvoiceMatchesByMoveIDs(
	ctx context.Context,
	keys []string,
) ([][]*carriersettlement.InvoiceMatch, error) {
	return loadercore.LoadAll(ctx, carrierInvoiceMatchesByMoveIDKey, keys)
}

func (f *CarrierInvoiceMatchesByMoveIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
