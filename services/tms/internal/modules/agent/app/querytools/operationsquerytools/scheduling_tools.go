package operationsquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment/recurringshipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/recurringshipmentports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/typeutils"
)

const fieldSeriesName = "name"

var recurringShipmentStatuses = []string{
	string(recurringshipment.StatusActive),
	string(recurringshipment.StatusPaused),
	string(recurringshipment.StatusExpired),
}

func Providers() []any {
	return []any{}
}

type recurringShipmentRow struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Status           string                 `json:"status"`
	CustomerID       string                 `json:"customerId,omitempty"`
	Customer         string                 `json:"customer,omitempty"`
	Lane             string                 `json:"lane,omitempty"`
	SourceShipmentID string                 `json:"sourceShipmentId"`
	Schedule         string                 `json:"schedule"`
	Timezone         string                 `json:"timezone"`
	AutoGenerate     bool                   `json:"autoGenerate"`
	NextOccurrenceAt querytool.OptionalDate `json:"nextOccurrenceAt"`
	GenerationCount  int64                  `json:"generationCount"`
	LastShipmentID   string                 `json:"lastGeneratedShipmentId,omitempty"`
}

func recurringShipmentRowFrom(entity *recurringshipment.RecurringShipment) recurringShipmentRow {
	row := recurringShipmentRow{
		ID:               entity.ID.String(),
		Name:             entity.Name,
		Status:           string(entity.Status),
		CustomerID:       querytool.PulidString(entity.CustomerID),
		SourceShipmentID: entity.SourceShipmentID.String(),
		Schedule:         entity.CronExpression,
		Timezone:         entity.Timezone,
		AutoGenerate:     entity.AutoGenerate,
		NextOccurrenceAt: querytool.ExpectedDate(
			typeutils.ValueOrZero(entity.NextOccurrenceAt),
			"none scheduled",
		),
		GenerationCount: entity.GenerationCount,
		LastShipmentID:  querytool.PulidString(entity.LastGeneratedShipmentID),
	}
	if entity.Customer != nil {
		row.Customer = entity.Customer.Name
	}
	if entity.OriginLocation != nil && entity.DestinationLocation != nil {
		row.Lane = entity.OriginLocation.Name + " to " + entity.DestinationLocation.Name
	}

	return row
}

func NewListRecurringShipmentsTool(
	repo recurringshipmentports.RecurringShipmentRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         "shipment_list_recurring",
		Row:          recurringShipmentRow{},
		EntityPlural: "recurring shipments",
		Summary: "List recurring shipments: the series that copy one saved shipment on a " +
			"schedule for a customer's standing lane, with when each next generates. Use " +
			"the id with shipment_update_recurring, shipment_set_recurring_status or " +
			"shipment_generate_recurring.",
		Resource: permission.ResourceRecurringShipment,
		Config: listquery.Describe(
			(*recurringshipment.RecurringShipment)(nil),
			recurringshipmentdb.RecurringShipmentsSchema(),
		),
		Fields: []filtercatalog.Field{
			{
				Name:   querytool.ParamStatus,
				Kind:   filtercatalog.KindEnum,
				Values: recurringShipmentStatuses,
				Note:   "Paused generates nothing until resumed; Expired has ended",
			},
			{Name: fieldSeriesName, Kind: filtercatalog.KindText, Sortable: true},
			{Name: "autoGenerate", Kind: filtercatalog.KindBoolean},
			{Name: "nextOccurrenceAt", Kind: filtercatalog.KindDate, Sortable: true},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(
				ctx,
				&recurringshipmentports.ListRecurringShipmentsRequest{Filter: opts},
			)
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(
				result.Items,
				func(item *recurringshipment.RecurringShipment) any {
					return recurringShipmentRowFrom(item)
				},
			), nil
		},
	})
}
