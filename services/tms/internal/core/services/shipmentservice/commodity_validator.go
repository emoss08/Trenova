package shipmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

type commodityRuleContext struct {
	index            int
	commodity        *shipment.ShipmentCommodity
	shipmentID       pulid.ID
	isCreate         bool
	seenCommodityIDs map[pulid.ID]string
	multiErr         *errortypes.MultiError
}

func createCommodityValidationRule() validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.
		NewTenantedRule[*shipment.Shipment]("shipment_commodity_validation").
		OnBoth().
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *shipment.Shipment,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			seenCommodityIDs := make(map[pulid.ID]string, len(entity.Commodities))

			for index, shipmentCommodity := range entity.Commodities {
				if shipmentCommodity == nil {
					continue
				}

				shipmentCommodity.Validate(multiErr.WithIndex("commodities", index))

				ruleCtx := commodityRuleContext{
					index:            index,
					commodity:        shipmentCommodity,
					shipmentID:       entity.ID,
					isCreate:         valCtx.IsCreate(),
					seenCommodityIDs: seenCommodityIDs,
					multiErr:         multiErr,
				}

				validateShipmentCommodityIdentifiers(ruleCtx)
			}

			return nil
		})
}

func validateShipmentCommodityIdentifiers(ctx commodityRuleContext) {
	if ctx.isCreate {
		if !ctx.commodity.ID.IsNil() {
			ctx.multiErr.Add(
				shipmentCommodityFieldPath(ctx.index, "id"),
				errortypes.ErrInvalid,
				"Shipment commodity ID must not be provided when creating a shipment",
			)
		}
		if !ctx.commodity.ShipmentID.IsNil() {
			ctx.multiErr.Add(
				shipmentCommodityFieldPath(ctx.index, "shipmentId"),
				errortypes.ErrInvalid,
				"Shipment commodity shipment ID must not be provided when creating a shipment",
			)
		}
	} else if !ctx.shipmentID.IsNil() &&
		!ctx.commodity.ShipmentID.IsNil() &&
		ctx.commodity.ShipmentID != ctx.shipmentID {
		ctx.multiErr.Add(
			shipmentCommodityFieldPath(ctx.index, "shipmentId"),
			errortypes.ErrInvalid,
			"Shipment commodity shipment ID must match the shipment being updated",
		)
	}

	if ctx.commodity.ID.IsNil() {
		return
	}

	currentPath := shipmentCommodityFieldPath(ctx.index, "id")
	if firstPath, ok := ctx.seenCommodityIDs[ctx.commodity.ID]; ok {
		ctx.multiErr.Add(
			currentPath,
			errortypes.ErrDuplicate,
			fmt.Sprintf("Shipment commodity ID duplicates %s", firstPath),
		)
		return
	}

	ctx.seenCommodityIDs[ctx.commodity.ID] = currentPath
}

func shipmentCommodityFieldPath(index int, field string) string {
	return fmt.Sprintf("commodities[%d].%s", index, field)
}
