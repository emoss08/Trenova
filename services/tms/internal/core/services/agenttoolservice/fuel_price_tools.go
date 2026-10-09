package agenttoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramFuelIndexID      = "fuelIndexId"
	paramFuelIndexPriceID = "fuelIndexPriceId"
	paramPriceDate        = "priceDate"
	paramPrice            = "price"
	fuelIndexSupplier     = "The custom fuel index, from get_fuel_surcharge_rates. Never guess one."
	fuelPriceSupplier     = "The manual price, from list_fuel_index_prices. Never guess one."
	fuelPriceRationale    = "Moves the price every fuel surcharge on this index is computed " +
		"from, which changes what customers are charged; only a person sets it."
)

type fuelIndexPriceKeeper interface {
	PlanAddManualPrice(
		ctx context.Context,
		entity *fuelsurcharge.FuelIndexPrice,
		userID pulid.ID,
	) (*fuelsurcharge.FuelIndexPrice, error)
	AddManualPrice(
		ctx context.Context,
		entity *fuelsurcharge.FuelIndexPrice,
		userID pulid.ID,
	) (*fuelsurcharge.FuelIndexPrice, error)
	PlanUpdateManualPrice(
		ctx context.Context,
		entity *fuelsurcharge.FuelIndexPrice,
		userID pulid.ID,
	) (*fuelsurchargeservice.PriceChange, error)
	UpdateManualPrice(
		ctx context.Context,
		entity *fuelsurcharge.FuelIndexPrice,
		userID pulid.ID,
	) (*fuelsurcharge.FuelIndexPrice, error)
}

var _ fuelIndexPriceKeeper = (*fuelsurchargeservice.Service)(nil)

type fuelIndexPriceView struct {
	PriceDate string `json:"priceDate"`
	Price     string `json:"price"`
	Currency  string `json:"currency"`
}

func fuelIndexPriceViewOf(price *fuelsurcharge.FuelIndexPrice) *fuelIndexPriceView {
	return &fuelIndexPriceView{
		PriceDate: price.PriceDate,
		Price:     price.Price.String(),
		Currency:  price.Currency,
	}
}

func fuelPriceMoneySpec(spec *receivableSpec) *receivableSpec {
	spec.resource = permission.ResourceFuelSurchargeProgram
	spec.operation = permission.OpUpdate
	spec.rationale = fuelPriceRationale

	return ledgerMoneySpec(spec)
}

func requirePriceDay(params map[string]any) (string, error) {
	raw, err := requireString(params, paramPriceDate)
	if err != nil {
		return "", err
	}
	day := strings.TrimSpace(raw)
	if _, err = time.Parse(fuelsurcharge.PriceDateLayout, day); err != nil {
		return "", fmt.Errorf("parameter %q must be YYYY-MM-DD, got %q", paramPriceDate, raw)
	}

	return day, nil
}

func fuelIndexPriceFrom(
	params *serviceports.ToolExecuteParams,
	idKey string,
) (*fuelsurcharge.FuelIndexPrice, error) {
	id, err := requirePulid(params.Params, idKey)
	if err != nil {
		return nil, err
	}
	day, err := requirePriceDay(params.Params)
	if err != nil {
		return nil, err
	}
	price, present, err := optionalDecimal(params.Params, paramPrice)
	if err != nil {
		return nil, err
	}
	if !present || !price.IsPositive() {
		return nil, fmt.Errorf("parameter %q must be a price greater than zero", paramPrice)
	}

	entity := &fuelsurcharge.FuelIndexPrice{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		PriceDate:      day,
		Price:          price,
	}
	if idKey == paramFuelIndexID {
		entity.FuelIndexID = id
	} else {
		entity.ID = id
	}

	return entity, nil
}

func fuelPriceProperties(idKey, idDescription string) map[string]any {
	return map[string]any{
		idKey: stringProperty(idDescription, 0),
		paramPriceDate: agenttoolschema.Date(
			"The day the price applies to, as the index publishes " +
				"it.",
		),
		paramPrice: stringProperty("The price per gallon as a decimal such as 3.899.", 0),
	}
}

func newRecordFuelIndexPriceTool(prices fuelIndexPriceKeeper) serviceports.AgentTool {
	return newReceivableTool(fuelPriceMoneySpec(&receivableSpec{
		name: "record_fuel_index_price",
		description: "Propose recording a published price on a custom fuel index, such as " +
			"a customer's own index or a rack price the organization tracks. Every fuel " +
			"surcharge that follows the index prices from it, so a person always decides. " +
			"EIA indexes are fetched on their own and take no manual price.",
		properties: fuelPriceProperties(paramFuelIndexID, fuelIndexSupplier),
		required:   []string{paramFuelIndexID, paramPriceDate, paramPrice},
		searchTerms: []string{
			"fuel price", "diesel price", "index", "surcharge", "rack", "doe",
		},
	}), receivablePlan[*fuelsurcharge.FuelIndexPrice, *fuelsurcharge.FuelIndexPrice]{
		request: func(params *serviceports.ToolExecuteParams) (*fuelsurcharge.FuelIndexPrice, error) {
			return fuelIndexPriceFrom(params, paramFuelIndexID)
		},
		plan: func(
			ctx context.Context,
			entity *fuelsurcharge.FuelIndexPrice,
			params *serviceports.ToolExecuteParams,
		) (*fuelsurcharge.FuelIndexPrice, error) {
			return prices.PlanAddManualPrice(ctx, entity, params.Actor.UserID)
		},
		refused: func(entity *fuelsurcharge.FuelIndexPrice) string {
			return fmt.Sprintf("Would record a fuel index price for %s.", entity.PriceDate)
		},
		render: func(
			_ *fuelsurcharge.FuelIndexPrice,
			planned *fuelsurcharge.FuelIndexPrice,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(toolpreview.Record{
				Resource: permission.ResourceFuelSurchargeProgram,
				Label:    "Fuel index price for " + planned.PriceDate,
			}, fuelIndexPriceViewOf(planned))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would record %s %s a gallon for %s on the fuel index; every surcharge "+
					"that follows it prices from this.",
				planned.Price.String(), planned.Currency, planned.PriceDate,
			), change), nil
		},
		run: func(
			ctx context.Context,
			entity *fuelsurcharge.FuelIndexPrice,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := prices.AddManualPrice(ctx, entity, params.Actor.UserID)

			return nil, err
		},
	})
}

func newCorrectFuelIndexPriceTool(prices fuelIndexPriceKeeper) serviceports.AgentTool {
	return newReceivableTool(fuelPriceMoneySpec(&receivableSpec{
		name:   "correct_fuel_index_price",
		recipe: []string{"list_fuel_index_prices", "correct_fuel_index_price"},
		description: "Propose correcting a manual price on a custom fuel index that was " +
			"entered wrong: its day or its price. A price fetched from EIA cannot be " +
			"corrected. Every surcharge that follows the index prices from it, so a person " +
			"always decides.",
		properties: fuelPriceProperties(paramFuelIndexPriceID, fuelPriceSupplier),
		required:   []string{paramFuelIndexPriceID, paramPriceDate, paramPrice},
	}), receivablePlan[*fuelsurcharge.FuelIndexPrice, *fuelsurchargeservice.PriceChange]{
		request: func(params *serviceports.ToolExecuteParams) (*fuelsurcharge.FuelIndexPrice, error) {
			return fuelIndexPriceFrom(params, paramFuelIndexPriceID)
		},
		plan: func(
			ctx context.Context,
			entity *fuelsurcharge.FuelIndexPrice,
			params *serviceports.ToolExecuteParams,
		) (*fuelsurchargeservice.PriceChange, error) {
			return prices.PlanUpdateManualPrice(ctx, entity, params.Actor.UserID)
		},
		refused: func(*fuelsurcharge.FuelIndexPrice) string {
			return "Would correct a fuel index price."
		},
		render: func(
			_ *fuelsurcharge.FuelIndexPrice,
			plan *fuelsurchargeservice.PriceChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(toolpreview.Record{
				Resource: permission.ResourceFuelSurchargeProgram,
				ID:       plan.Before.ID,
				Label:    "Fuel index price for " + plan.Before.PriceDate,
			}, fuelIndexPriceViewOf(plan.Before), fuelIndexPriceViewOf(plan.After))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would change the fuel index price for %s from %s to %s %s a gallon; every "+
					"surcharge that follows it prices from this.",
				plan.Before.PriceDate, plan.Before.Price.String(), plan.After.Price.String(),
				plan.After.Currency,
			), change), nil
		},
		run: func(
			ctx context.Context,
			entity *fuelsurcharge.FuelIndexPrice,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := prices.UpdateManualPrice(ctx, entity, params.Actor.UserID)

			return nil, err
		},
	})
}
