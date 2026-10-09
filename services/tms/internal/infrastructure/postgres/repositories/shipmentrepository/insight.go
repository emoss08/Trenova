package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tableinsight"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type InsightParams struct {
	fx.In

	DB           *postgres.Connection
	QuickFilters services.ShipmentQuickFilterBasisResolver
}

type insightSource struct {
	quickFilters services.ShipmentQuickFilterBasisResolver
}

type insightOptions struct {
	Status                  string                     `json:"status"`
	ActivityWindowStart     int64                      `json:"activityWindowStart"`
	ActivityWindowEnd       int64                      `json:"activityWindowEnd"`
	BillingTransferEligible bool                       `json:"billingTransferEligible"`
	QuickFilters            []shipment.QuickFilterSpec `json:"quickFilters"`
	Timezone                string                     `json:"timezone"`
}

func NewInsightSource(p InsightParams) tableinsight.Result {
	cols := buncolgen.ShipmentColumns
	facet := func(column buncolgen.Column) dbhelper.InsightColumn {
		return dbhelper.InsightColumn{Column: column, Facetable: true}
	}
	sum := func(column buncolgen.Column) dbhelper.InsightColumn {
		return dbhelper.InsightColumn{Column: column, Summable: true}
	}
	timeline := func(column buncolgen.Column) dbhelper.InsightColumn {
		return dbhelper.InsightColumn{Column: column, Timeline: true}
	}
	source := &insightSource{quickFilters: p.QuickFilters}

	return tableinsight.New(&tableinsight.Config{
		Resource: permission.ResourceShipment,
		DB:       p.DB,
		IDColumn: cols.ID,
		Base:     source.baseQuery,
		Columns: dbhelper.InsightColumns{
			"status":                facet(cols.Status),
			"tenderStatus":          facet(cols.TenderStatus),
			"billingTransferStatus": facet(cols.BillingTransferStatus),
			"customerId":            facet(cols.CustomerID),
			"billToCustomerId":      facet(cols.BillToCustomerID),
			"shipmentTypeId":        facet(cols.ShipmentTypeID),
			"serviceTypeId":         facet(cols.ServiceTypeID),
			"tractorTypeId":         facet(cols.TractorTypeID),
			"trailerTypeId":         facet(cols.TrailerTypeID),
			"ownerId":               facet(cols.OwnerID),
			"enteredById":           facet(cols.EnteredByID),
			"formulaTemplateId":     facet(cols.FormulaTemplateID),
			"rateAgreementId":       facet(cols.RateAgreementID),
			"totalChargeAmount":     sum(cols.TotalChargeAmount),
			"freightChargeAmount":   sum(cols.FreightChargeAmount),
			"otherChargeAmount":     sum(cols.OtherChargeAmount),
			"pieces":                sum(cols.Pieces),
			"weight":                sum(cols.Weight),
			"createdAt":             timeline(cols.CreatedAt),
			"actualShipDate":        timeline(cols.ActualShipDate),
			"actualDeliveryDate":    timeline(cols.ActualDeliveryDate),
			"billedAt":              timeline(cols.BilledAt),
		},
	})
}

func (s *insightSource) baseQuery(
	ctx context.Context,
	dba bun.IDB,
	scope *repositories.TableInsightScope,
) (*bun.SelectQuery, error) {
	options := new(insightOptions)
	if len(scope.Options) > 0 {
		if err := jsonutils.Convert(scope.Options, options); err != nil {
			return nil, errortypes.NewValidationError(
				"options",
				errortypes.ErrInvalid,
				"The shipment board options are not in a shape this table reads",
			)
		}
	}

	req := &repositories.ListShipmentsRequest{
		Filter: scope.Filter,
		ShipmentOptions: repositories.ShipmentOptions{
			Status:                  options.Status,
			ActivityWindowStart:     options.ActivityWindowStart,
			ActivityWindowEnd:       options.ActivityWindowEnd,
			BillingTransferEligible: options.BillingTransferEligible,
			QuickFilters:            options.QuickFilters,
			Timezone:                options.Timezone,
		},
	}

	if scope.Filter != nil {
		if err := s.quickFilters.Prepare(
			ctx,
			scope.Filter.TenantInfo,
			&req.ShipmentOptions,
		); err != nil {
			return nil, err
		}
	}

	quick, err := QuickFilterConditions(dba, &req.ShipmentOptions)
	if err != nil {
		return nil, err
	}

	return dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return countShipmentListQuery(sq, dba, req).Apply(whereAll(quick))
		}), nil
}
