package recurringshipmentresolver

import (
	"math"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/recurringshipmentports"
)

const (
	defaultRunsLimit = 50
	maxRunsLimit     = 100
)

func runsLimit(first *int) int {
	if first == nil || *first <= 0 {
		return defaultRunsLimit
	}
	return min(*first, maxRunsLimit)
}

func matchRequestFromInput(
	input *gqlmodel.MatchRecurringShipmentsInput,
) (*recurringshipmentports.MatchRecurringShipmentsRequest, error) {
	customerID, err := gqlid.ParseIDField("customerId", input.CustomerID)
	if err != nil {
		return nil, err
	}
	originID, err := gqlid.ParseIDField("originLocationId", input.OriginLocationID)
	if err != nil {
		return nil, err
	}
	destinationID, err := gqlid.ParseIDField("destinationLocationId", input.DestinationLocationID)
	if err != nil {
		return nil, err
	}

	return &recurringshipmentports.MatchRecurringShipmentsRequest{
		CustomerID:            customerID,
		OriginLocationID:      originID,
		DestinationLocationID: destinationID,
	}, nil
}

func applyRecurringShipmentInput(
	entity *recurringshipment.RecurringShipment,
	input *gqlmodel.RecurringShipmentInput,
) error {
	sourceShipmentID, err := gqlid.ParseIDField("sourceShipmentId", input.SourceShipmentID)
	if err != nil {
		return err
	}

	entity.SourceShipmentID = sourceShipmentID
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	if input.Status != nil {
		entity.Status = *input.Status
	}
	entity.CronExpression = input.CronExpression
	entity.Timezone = input.Timezone
	entity.StartDate = gqlscalar.Int64Value(input.StartDate)
	entity.EndDate = gqlscalar.Int64Ptr(input.EndDate)
	entity.MaxOccurrences = clampedInt32Ptr(input.MaxOccurrences)
	entity.LeadTimeDays = clampedInt16(input.LeadTimeDays)
	entity.SkipWeekends = input.SkipWeekends
	if input.ExceptionPolicy != nil {
		entity.ExceptionPolicy = *input.ExceptionPolicy
	}
	entity.BlackoutDates = input.BlackoutDates
	entity.AutoGenerate = input.AutoGenerate
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return nil
}

func clampedInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	clamped := int32(max(min(*value, math.MaxInt32), math.MinInt32))
	return &clamped
}

func clampedInt16(value int) int16 {
	return int16(max(min(value, math.MaxInt16), math.MinInt16))
}
