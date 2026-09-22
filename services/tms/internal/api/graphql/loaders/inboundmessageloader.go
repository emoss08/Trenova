package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type inboundAttachmentCounter interface {
	CountAttachmentsByMessageIDs(
		ctx context.Context,
		req repositories.CountInboundAttachmentsRequest,
	) (map[pulid.ID]int, error)
}

type InboundAttachmentCountLoaderFactoryParams struct {
	fx.In

	MessageRepo repositories.InboundMessageRepository
}

type InboundAttachmentCountLoaderFactory struct {
	messages inboundAttachmentCounter
}

func NewInboundAttachmentCountLoaderFactory(
	p InboundAttachmentCountLoaderFactoryParams,
) *InboundAttachmentCountLoaderFactory {
	return &InboundAttachmentCountLoaderFactory{messages: p.MessageRepo}
}

func (f *InboundAttachmentCountLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InboundAttachmentCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.messages.CountAttachmentsByMessageIDs(ctx, repositories.CountInboundAttachmentsRequest{
			TenantInfo: tenantInfo,
			MessageIDs: ids,
		})
	})
}

type shipmentSummaryLister interface {
	ListSummariesByIDs(
		ctx context.Context,
		req *repositories.ListShipmentSummariesRequest,
	) ([]*repositories.ShipmentSummary, error)
}

type ShipmentSummaryByIDLoaderFactoryParams struct {
	fx.In

	ShipmentRepo repositories.ShipmentRepository
}

type ShipmentSummaryByIDLoaderFactory struct {
	shipments shipmentSummaryLister
}

func NewShipmentSummaryByIDLoaderFactory(
	p ShipmentSummaryByIDLoaderFactoryParams,
) *ShipmentSummaryByIDLoaderFactory {
	return &ShipmentSummaryByIDLoaderFactory{shipments: p.ShipmentRepo}
}

func (f *ShipmentSummaryByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *repositories.ShipmentSummary] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentSummaryByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*repositories.ShipmentSummary] {
	return batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*repositories.ShipmentSummary, error) {
			return f.shipments.ListSummariesByIDs(ctx, &repositories.ListShipmentSummariesRequest{
				TenantInfo:  tenantInfo,
				ShipmentIDs: ids,
			})
		},
		"Shipment not found within your organization",
	)
}
