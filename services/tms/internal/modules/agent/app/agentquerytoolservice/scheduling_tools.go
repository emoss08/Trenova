package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/recurringshipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/typeutils"
)

const fieldSeriesName = "name"

var recurringShipmentStatuses = []string{
	string(recurringshipment.StatusActive),
	string(recurringshipment.StatusPaused),
	string(recurringshipment.StatusExpired),
}

func operationsQueryToolProviders() []any {
	return []any{
		newListRecurringShipmentsTool,
	}
}

type recurringShipmentRow struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Status           string       `json:"status"`
	CustomerID       string       `json:"customerId,omitempty"`
	Customer         string       `json:"customer,omitempty"`
	Lane             string       `json:"lane,omitempty"`
	SourceShipmentID string       `json:"sourceShipmentId"`
	Schedule         string       `json:"schedule"`
	Timezone         string       `json:"timezone"`
	AutoGenerate     bool         `json:"autoGenerate"`
	NextOccurrenceAt optionalDate `json:"nextOccurrenceAt"`
	GenerationCount  int64        `json:"generationCount"`
	LastShipmentID   string       `json:"lastGeneratedShipmentId,omitempty"`
}

func recurringShipmentRowFrom(entity *recurringshipment.RecurringShipment) recurringShipmentRow {
	row := recurringShipmentRow{
		ID:               entity.ID.String(),
		Name:             entity.Name,
		Status:           string(entity.Status),
		CustomerID:       pulidString(entity.CustomerID),
		SourceShipmentID: entity.SourceShipmentID.String(),
		Schedule:         entity.CronExpression,
		Timezone:         entity.Timezone,
		AutoGenerate:     entity.AutoGenerate,
		NextOccurrenceAt: expectedDate(
			typeutils.ValueOrZero(entity.NextOccurrenceAt),
			"none scheduled",
		),
		GenerationCount: entity.GenerationCount,
		LastShipmentID:  pulidString(entity.LastGeneratedShipmentID),
	}
	if entity.Customer != nil {
		row.Customer = entity.Customer.Name
	}
	if entity.OriginLocation != nil && entity.DestinationLocation != nil {
		row.Lane = entity.OriginLocation.Name + " to " + entity.DestinationLocation.Name
	}

	return row
}

func newListRecurringShipmentsTool(
	repo recurringshipmentports.RecurringShipmentRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "shipment_list_recurring",
		entityPlural: "recurring shipments",
		summary: "List recurring shipments: the series that copy one saved shipment on a " +
			"schedule for a customer's standing lane, with when each next generates. Use " +
			"the id with shipment_update_recurring, shipment_set_recurring_status or " +
			"shipment_generate_recurring.",
		resource: permission.ResourceRecurringShipment,
		config:   querybuilder.GetFieldConfiguration((*recurringshipment.RecurringShipment)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: recurringShipmentStatuses,
				Note:   "Paused generates nothing until resumed; Expired has ended",
			},
			{Name: fieldSeriesName, Kind: filterText, Sortable: true},
			{Name: "autoGenerate", Kind: filterBool},
			{Name: "nextOccurrenceAt", Kind: filterDate, Sortable: true},
			{Name: fieldCreatedAt, Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listPage, error) {
			result, err := repo.List(
				ctx,
				&recurringshipmentports.ListRecurringShipmentsRequest{Filter: opts},
			)
			if err != nil {
				return listPage{}, err
			}

			return offsetListPage(
				result.Items,
				func(item *recurringshipment.RecurringShipment) any {
					return recurringShipmentRowFrom(item)
				},
			), nil
		},
	})
}
