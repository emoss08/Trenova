package billingclientprovider

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

var _ services.AnalyticsPageProvider = (*Provider)(nil)

type ProviderParams struct {
	fx.In

	Logger *logger.Logger
	DB     db.Connection
}

type Provider struct {
	l  *zerolog.Logger
	db db.Connection
}

func NewProvider(p ProviderParams) *Provider {
	log := p.Logger.With().
		Str("provider", "billing_client_analytics").
		Logger()

	return &Provider{
		l:  &log,
		db: p.DB,
	}
}

func (p *Provider) GetPage() services.AnalyticsPage {
	return services.BillingClientAnalyticsPage
}

func (p *Provider) GetAnalyticsData(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (services.AnalyticsData, error) {
	log := p.l.With().
		Str("operation", "GetAnalyticsData").
		Str("orgID", opts.OrgID.String()).
		Str("buID", opts.BuID.String()).
		Str("userID", opts.UserID.String()).
		Logger()

	shipmentsReadyToBill, err := p.GetShipmentsReadyToBill(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error().Err(err).Msg("get shipments ready to bill")
		return services.AnalyticsData{}, err
	}

	completedShipmentsWithNoDocuments, err := p.GetCompletedShipmentsWithNoDocuments(
		ctx,
		opts.OrgID,
		opts.BuID,
	)
	if err != nil {
		log.Error().Err(err).Msg("get completed shipments with no documents")
		return services.AnalyticsData{}, err
	}

	return services.AnalyticsData{
		"shipmentReadyBillCard":             shipmentsReadyToBill,
		"completedShipmentsWithNoDocuments": completedShipmentsWithNoDocuments,
	}, nil
}

func (p *Provider) GetShipmentsReadyToBill(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*ShipmentReadyToBillCard, error) {
	log := p.l.With().
		Str("query", "getShipmentsReadyToBill").
		Str("orgID", orgID.String()).
		Str("buID", buID.String()).
		Logger()

	dba, err := p.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("get database connection")
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	count, err := dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sp.status = ?", shipment.StatusReadyToBill).
				Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID)
		}).
		Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("get shipments ready to bill")
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get shipments ready to bill")
	}

	return &ShipmentReadyToBillCard{
		Count: count,
	}, nil
}

func (p *Provider) GetCompletedShipmentsWithNoDocuments(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*CompletedShipmentsWithNoDocumentsCard, error) {
	log := p.l.With().
		Str("query", "getCompletedShipmentsWithNoDocuments").
		Str("orgID", orgID.String()).
		Str("buID", buID.String()).
		Logger()

	dba, err := p.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("get database connection")
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	count, err := dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Where("sp.status = ?", shipment.StatusCompleted).
		Where("sp.organization_id = ?", orgID).
		Where("sp.business_unit_id = ?", buID).
		Where("NOT EXISTS (?)", dba.NewSelect().
			Model((*document.Document)(nil)).
			ColumnExpr("1"). // Select 1 for existence check
			Where("doc.resource_id = sp.id").
			Where("doc.resource_type = ?", permission.ResourceShipment).
			Where("doc.organization_id = sp.organization_id").
			Where("doc.business_unit_id = sp.business_unit_id"),
		).
		Count(ctx)

	if err != nil {
		log.Error().Err(err).Msg("get completed shipments with no documents")
		return nil, oops.
			In("Provider.GetCompletedShipmentsWithNoDocuments").
			Time(time.Now()).
			Wrapf(err, "failed to count completed shipments with no documents")
	}

	return &CompletedShipmentsWithNoDocumentsCard{
		Count: count,
	}, nil
}
