package documentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type inboundAttachmentCounter interface {
	CountAttachmentsByMessageIDs(
		ctx context.Context,
		req inboundmessageports.CountInboundAttachmentsRequest,
	) (map[pulid.ID]int, error)
}

type InboundAttachmentCountLoaderFactoryParams struct {
	fx.In

	MessageRepo inboundmessageports.InboundMessageRepository
}

type InboundAttachmentCountLoaderFactory struct {
	messages inboundAttachmentCounter
}

func NewInboundAttachmentCountLoaderFactory(
	p InboundAttachmentCountLoaderFactoryParams,
) *InboundAttachmentCountLoaderFactory {
	return &InboundAttachmentCountLoaderFactory{messages: p.MessageRepo}
}

func newInboundAttachmentCountLoader(f *InboundAttachmentCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *InboundAttachmentCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[int] {
	return loadercore.BatchCountFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
			return f.messages.CountAttachmentsByMessageIDs(
				ctx,
				inboundmessageports.CountInboundAttachmentsRequest{
					TenantInfo: tenantInfo,
					MessageIDs: ids,
				},
			)
		},
	)
}

type shipmentSummaryLister interface {
	ListSummariesByIDs(
		ctx context.Context,
		req *shipmentports.ListShipmentSummariesRequest,
	) ([]*shipmentports.ShipmentSummary, error)
}

type ShipmentSummaryByIDLoaderFactoryParams struct {
	fx.In

	ShipmentRepo shipmentports.ShipmentRepository
}

type ShipmentSummaryByIDLoaderFactory struct {
	shipments shipmentSummaryLister
}

func NewShipmentSummaryByIDLoaderFactory(
	p ShipmentSummaryByIDLoaderFactoryParams,
) *ShipmentSummaryByIDLoaderFactory {
	return &ShipmentSummaryByIDLoaderFactory{shipments: p.ShipmentRepo}
}

func newShipmentSummaryByIDLoader(f *ShipmentSummaryByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *shipmentports.ShipmentSummary] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentSummaryByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*shipmentports.ShipmentSummary] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*shipmentports.ShipmentSummary, error) {
			return f.shipments.ListSummariesByIDs(ctx, &shipmentports.ListShipmentSummariesRequest{
				TenantInfo:  tenantInfo,
				ShipmentIDs: ids,
			})
		},
		"Shipment not found within your organization",
	)
}

var (
	inboundAttachmentCountKey = loadercore.NewKey(newInboundAttachmentCountLoader)
	shipmentSummaryByIDKey    = loadercore.NewKey(newShipmentSummaryByIDLoader)
)

func InboundAttachmentCount(ctx context.Context, key string) (int, error) {
	return loadercore.Load(ctx, inboundAttachmentCountKey, key)
}

func (f *InboundAttachmentCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func ShipmentSummaryByID(ctx context.Context, key string) (*shipmentports.ShipmentSummary, error) {
	return loadercore.Load(ctx, shipmentSummaryByIDKey, key)
}

func (f *ShipmentSummaryByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
