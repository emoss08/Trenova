package loaders

import (
	"context"

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
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InboundAttachmentCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.messages.CountAttachmentsByMessageIDs(
			ctx,
			inboundmessageports.CountInboundAttachmentsRequest{
				TenantInfo: tenantInfo,
				MessageIDs: ids,
			},
		)
	})
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
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentSummaryByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*shipmentports.ShipmentSummary] {
	return batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*shipmentports.ShipmentSummary, error) {
			return f.shipments.ListSummariesByIDs(ctx, &shipmentports.ListShipmentSummariesRequest{
				TenantInfo:  tenantInfo,
				ShipmentIDs: ids,
			})
		},
		"Shipment not found within your organization",
	)
}
