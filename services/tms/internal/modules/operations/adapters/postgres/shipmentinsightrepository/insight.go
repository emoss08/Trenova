package shipmentinsightrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tableinsight"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentpredicate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
)

type InsightParams struct {
	fx.In

	DB           *postgres.Connection
	QuickFilters shipmentports.ShipmentQuickFilterBasisResolver
}

type insightSource struct {
	quickFilters shipmentports.ShipmentQuickFilterBasisResolver
}

type insightOptions struct {
	Status                  string                        `json:"status"`
	ActivityWindowStart     int64                         `json:"activityWindowStart"`
	ActivityWindowEnd       int64                         `json:"activityWindowEnd"`
	BillingTransferEligible bool                          `json:"billingTransferEligible"`
	QuickFilters            []quickfilter.QuickFilterSpec `json:"quickFilters"`
	Timezone                string                        `json:"timezone"`
}

func NewInsightSource(p InsightParams) tableinsight.Result {
	sp := shipmentdb.Shipments
	source := &insightSource{quickFilters: p.QuickFilters}

	return tableinsight.New(&tableinsight.Config{
		Resource: permission.ResourceShipment,
		DB:       p.DB,
		IDColumn: sp.ID.Col,
		Base:     source.baseQuery,
		Columns: tableinsight.Columns{
			"status":                tableinsight.Facet(sp.Status.Col),
			"tenderStatus":          tableinsight.Facet(sp.TenderStatus.Col),
			"billingTransferStatus": tableinsight.Facet(sp.BillingTransferStatus.Col),
			"customerId":            tableinsight.Facet(sp.CustomerID.Col),
			"billToCustomerId":      tableinsight.Facet(sp.BillToCustomerID.Col),
			"shipmentTypeId":        tableinsight.Facet(sp.ShipmentTypeID.Col),
			"serviceTypeId":         tableinsight.Facet(sp.ServiceTypeID.Col),
			"tractorTypeId":         tableinsight.Facet(sp.TractorTypeID.Col),
			"trailerTypeId":         tableinsight.Facet(sp.TrailerTypeID.Col),
			"ownerId":               tableinsight.Facet(sp.OwnerID.Col),
			"enteredById":           tableinsight.Facet(sp.EnteredByID.Col),
			"formulaTemplateId":     tableinsight.Facet(sp.FormulaTemplateID.Col),
			"rateAgreementId":       tableinsight.Facet(sp.RateAgreementID.Col),
			"totalChargeAmount":     tableinsight.Sum(sp.TotalChargeAmount.Col),
			"freightChargeAmount":   tableinsight.Sum(sp.FreightChargeAmount.Col),
			"otherChargeAmount":     tableinsight.Sum(sp.OtherChargeAmount.Col),
			"pieces":                tableinsight.Sum(sp.Pieces.Col),
			"weight":                tableinsight.Sum(sp.Weight.Col),
			"createdAt":             tableinsight.Timeline(sp.CreatedAt.Col),
			"actualShipDate":        tableinsight.Timeline(sp.ActualShipDate.Col),
			"actualDeliveryDate":    tableinsight.Timeline(sp.ActualDeliveryDate.Col),
			"billedAt":              tableinsight.Timeline(sp.BilledAt.Col),
		},
	})
}

func (s *insightSource) baseQuery(
	ctx context.Context,
	scope *permissionports.TableInsightScope,
) (tableinsight.Base, error) {
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

	opts := shipmentports.ShipmentOptions{
		Status:                  options.Status,
		ActivityWindowStart:     options.ActivityWindowStart,
		ActivityWindowEnd:       options.ActivityWindowEnd,
		BillingTransferEligible: options.BillingTransferEligible,
		QuickFilters:            options.QuickFilters,
		Timezone:                options.Timezone,
	}

	if scope.Filter != nil {
		if err := s.quickFilters.Prepare(ctx, scope.Filter.TenantInfo, &opts); err != nil {
			return nil, err
		}
	}

	quick, err := shipmentpredicate.QuickFilterConditions(&opts)
	if err != nil {
		return nil, err
	}

	return tableinsight.Typed(shipmentpredicate.Scope(scope.Filter, &opts, quick)), nil
}
