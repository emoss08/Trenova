package partyquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
)

type carrierRow struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	CarrierType      string `json:"carrierType,omitempty"`
	ComplianceStatus string `json:"complianceStatus,omitempty"`
	SafetyRating     string `json:"safetyRating,omitempty"`
	DOTNumber        string `json:"dotNumber,omitempty"`
	MCNumber         string `json:"mcNumber,omitempty"`
}

func NewListCarriersTool(repo carrierports.CarrierRepository) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         querytool.ToolCarrierList,
		Row:          carrierRow{},
		EntityPlural: pluralCarriers,
		Summary:      carriersSummary,
		Resource:     permission.ResourceCarrier,
		Config:       listquery.Describe((*carrier.Carrier)(nil), carrierdb.CarriersSchema()),
		Fields: []filtercatalog.Field{
			{
				Name:   querytool.ListFieldStatus,
				Kind:   filtercatalog.KindEnum,
				Values: querytool.StatusValues,
			},
			{
				Name:   "complianceStatus",
				Kind:   filtercatalog.KindEnum,
				Values: []string{"Pending", "Qualified", "Disqualified", "Expired"},
				Note:   noteQualifiedCarrier,
			},
			{
				Name:   "safetyRating",
				Kind:   filtercatalog.KindEnum,
				Values: []string{"Satisfactory", "Conditional", "Unsatisfactory", "NotRated"},
			},
			{Name: querytool.ListFieldCode, Kind: filtercatalog.KindText, Sortable: true},
			{Name: querytool.ListFieldName, Kind: filtercatalog.KindText, Sortable: true},
			{Name: "dotNumber", Kind: filtercatalog.KindText},
			{Name: "mcNumber", Kind: filtercatalog.KindText},
			{
				Name:   "carrierType",
				Kind:   filtercatalog.KindEnum,
				Values: []string{"Common", "Contract", "Broker", "Exempt"},
			},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &carrierports.ListCarrierRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(result.Items, func(item *carrier.Carrier) any {
				return carrierRow{
					ID:               item.ID.String(),
					Code:             item.Code,
					Name:             item.Name,
					Status:           string(item.Status),
					CarrierType:      string(item.CarrierType),
					ComplianceStatus: string(item.ComplianceStatus),
					SafetyRating:     string(item.SafetyRating),
					DOTNumber:        item.DOTNumber,
					MCNumber:         item.MCNumber,
				}
			}), nil
		},
	})
}
