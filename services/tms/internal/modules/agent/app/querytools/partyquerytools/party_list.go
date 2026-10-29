package partyquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
)

type customerRow struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	City      string `json:"city,omitempty"`
	DOTNumber string `json:"dotNumber,omitempty"`
	MCNumber  string `json:"mcNumber,omitempty"`
}

func NewListCustomersTool(repo customerports.CustomerRepository) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         querytool.ToolCustomerList,
		Row:          customerRow{},
		EntityPlural: pluralCustomers,
		Summary:      customersSummary,
		Resource:     permission.ResourceCustomer,
		Config:       listquery.Describe((*customer.Customer)(nil), customerdb.CustomersSchema()),
		Fields: []filtercatalog.Field{
			{
				Name:   querytool.ListFieldStatus,
				Kind:   filtercatalog.KindEnum,
				Values: querytool.StatusValues,
			},
			{Name: querytool.ListFieldCode, Kind: filtercatalog.KindText, Sortable: true},
			{Name: querytool.ListFieldName, Kind: filtercatalog.KindText, Sortable: true},
			{Name: "city", Kind: filtercatalog.KindText, Sortable: true},
			{Name: "dotNumber", Kind: filtercatalog.KindText},
			{Name: "mcNumber", Kind: filtercatalog.KindText},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &customerports.ListCustomerRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.ResultPage(result, func(item *customer.Customer) any {
				return customerRow{
					ID:        item.ID.String(),
					Code:      item.Code,
					Name:      item.Name,
					Status:    string(item.Status),
					City:      item.City,
					DOTNumber: item.DOTNumber,
					MCNumber:  item.MCNumber,
				}
			}), nil
		},
	})
}

type locationRow struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Address string `json:"address,omitempty"`
	City    string `json:"city,omitempty"`
}

func NewListLocationsTool(repo locationports.LocationRepository) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         querytool.ToolLocationList,
		Row:          locationRow{},
		EntityPlural: pluralLocations,
		Summary:      locationsSummary,
		Resource:     permission.ResourceLocation,
		Config:       listquery.Describe((*location.Location)(nil), locationdb.LocationsSchema()),
		Fields: []filtercatalog.Field{
			{
				Name:   querytool.ListFieldStatus,
				Kind:   filtercatalog.KindEnum,
				Values: querytool.StatusValues,
			},
			{Name: querytool.ListFieldCode, Kind: filtercatalog.KindText, Sortable: true},
			{Name: querytool.ListFieldName, Kind: filtercatalog.KindText, Sortable: true},
			{Name: "city", Kind: filtercatalog.KindText, Sortable: true},
			{Name: "postalCode", Kind: filtercatalog.KindText},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &locationports.ListLocationRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.ResultPage(result, func(item *location.Location) any {
				return locationRow{
					ID:      item.ID.String(),
					Code:    item.Code,
					Name:    item.Name,
					Status:  string(item.Status),
					Address: item.AddressLine1,
					City:    item.City,
				}
			}), nil
		},
	})
}
