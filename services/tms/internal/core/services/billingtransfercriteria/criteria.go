package billingtransfercriteria

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ParamQuery              = "query"
	ParamStatus             = "status"
	ParamCustomerID         = "customerId"
	ParamDeliveredFrom      = "deliveredFrom"
	ParamDeliveredTo        = "deliveredTo"
	fieldActualDeliveryDate = "actualDeliveryDate"
	labelCustomer           = "customer"
	labelDeliveredFrom      = "delivered from"
	labelDeliveredTo        = "delivered to"
)

var Statuses = agenttoolschema.Source(
	"billingTransfer.candidateStatus",
	[]string{string(shipment.StatusReadyToInvoice), string(shipment.StatusCompleted)},
)

var paramNames = []string{
	ParamQuery,
	ParamStatus,
	ParamCustomerID,
	ParamDeliveredFrom,
	ParamDeliveredTo,
}

type Criteria struct {
	Query            string
	Status           shipment.Status
	CustomerID       pulid.ID
	DeliveredFrom    int64
	DeliveredTo      int64
	DeliveredFromDay string
	DeliveredToDay   string
}

func Properties() map[string]any {
	return map[string]any{
		ParamQuery: stringProperty("Words matched against pro number, BOL and the other " +
			"searchable columns, as the dialog's search box does."),
		ParamStatus: agenttoolschema.Enum(
			"Only shipments in this status. Completed shipments transfer only when "+
				"markCompletedReadyToInvoice is true.",
			Statuses,
		),
		ParamCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
			"Only this customer's shipments, by id from list_customers."),
		ParamDeliveredFrom: stringProperty("Only shipments delivered on or after this day. " +
			"A date as YYYY-MM-DD, or today."),
		ParamDeliveredTo: stringProperty("Only shipments delivered on or before this day. " +
			"A date as YYYY-MM-DD, or today."),
	}
}

func IsParam(name string) bool {
	return slices.Contains(paramNames, name)
}

func Given(params map[string]any) []string {
	given := make([]string, 0, len(paramNames))
	for _, name := range paramNames {
		if maputils.StringValue(params, name) != "" {
			given = append(given, name)
		}
	}

	return given
}

func Read(params map[string]any, clk filtercatalog.Clock) (Criteria, error) {
	criteria := Criteria{Query: maputils.StringValue(params, ParamQuery)}

	if status := maputils.StringValue(params, ParamStatus); status != "" {
		if !slices.Contains(Statuses.Values, status) {
			return Criteria{}, fmt.Errorf("%s %q is not one of %s",
				ParamStatus, status, strings.Join(Statuses.Values, ", "))
		}
		criteria.Status = shipment.Status(status)
	}

	if raw := maputils.StringValue(params, ParamCustomerID); raw != "" {
		id, err := pulid.Parse(raw)
		if err != nil {
			return Criteria{}, fmt.Errorf("parameter %q is not an id", ParamCustomerID)
		}
		criteria.CustomerID = id
	}

	var err error
	criteria.DeliveredFromDay, criteria.DeliveredFrom, err = readDay(
		params,
		ParamDeliveredFrom,
		clk,
	)
	if err != nil {
		return Criteria{}, err
	}
	criteria.DeliveredToDay, criteria.DeliveredTo, err = readDay(params, ParamDeliveredTo, clk)
	if err != nil {
		return Criteria{}, err
	}
	if criteria.DeliveredTo > 0 {
		criteria.DeliveredTo = clk.DayEnd(criteria.DeliveredTo)
	}
	if criteria.DeliveredFrom > 0 && criteria.DeliveredTo > 0 &&
		criteria.DeliveredFrom > criteria.DeliveredTo {
		return Criteria{}, fmt.Errorf("%s is after %s", ParamDeliveredFrom, ParamDeliveredTo)
	}

	return criteria, nil
}

func (c *Criteria) FieldFilters() []domaintypes.FieldFilter {
	filters := make([]domaintypes.FieldFilter, 0, 3)
	if c.CustomerID.IsNotNil() {
		filters = append(filters, domaintypes.FieldFilter{
			Field: ParamCustomerID, Operator: dbtype.OpEqual, Value: c.CustomerID.String(),
		})
	}
	if c.DeliveredFrom > 0 {
		filters = append(filters, domaintypes.FieldFilter{
			Field:    fieldActualDeliveryDate,
			Operator: dbtype.OpGreaterThanOrEqual,
			Value:    c.DeliveredFrom,
		})
	}
	if c.DeliveredTo > 0 {
		filters = append(filters, domaintypes.FieldFilter{
			Field:    fieldActualDeliveryDate,
			Operator: dbtype.OpLessThanOrEqual,
			Value:    c.DeliveredTo,
		})
	}

	return filters
}

func (c *Criteria) QueryOptions(
	tenant pagination.TenantInfo,
	page pagination.Info,
) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo:   tenant,
		Pagination:   page,
		Query:        c.Query,
		FieldFilters: c.FieldFilters(),
	}
}

func (c *Criteria) Describe(into *filtercatalog.Criteria) {
	into.Text(c.Query)
	if c.Status != "" {
		into.Field(ParamStatus, string(c.Status))
	}
	if c.CustomerID.IsNotNil() {
		into.Field(labelCustomer, c.CustomerID.String())
	}
	if c.DeliveredFromDay != "" {
		into.Field(labelDeliveredFrom, c.DeliveredFromDay)
	}
	if c.DeliveredToDay != "" {
		into.Field(labelDeliveredTo, c.DeliveredToDay)
	}
}

func readDay(
	params map[string]any,
	key string,
	clk filtercatalog.Clock,
) (raw string, day int64, err error) {
	raw = maputils.StringValue(params, key)
	if raw == "" {
		return "", 0, nil
	}
	day, ok := clk.Day(raw)
	if !ok {
		return "", 0, fmt.Errorf("parameter %q must be a date as YYYY-MM-DD", key)
	}

	return raw, day, nil
}

func stringProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
}
