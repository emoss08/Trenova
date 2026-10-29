package equipmentquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
)

var (
	equipmentStatuses = []string{"Available", "OutOfService", "AtMaintenance", "Sold"}
	ownershipTypes    = []string{"CompanyOwned", "Leased", "OwnerOperator"}
)

type equipmentRow struct {
	ID                 string                 `json:"id"`
	Code               string                 `json:"code"`
	Status             string                 `json:"status"`
	Make               string                 `json:"make,omitempty"`
	Model              string                 `json:"model,omitempty"`
	Year               int                    `json:"year,omitempty"`
	LicensePlate       string                 `json:"licensePlate,omitempty"`
	OwnershipType      string                 `json:"ownershipType,omitempty"`
	RegistrationExpiry querytool.OptionalDate `json:"registrationExpiry"`
	LastInspectionDate querytool.OptionalDate `json:"lastInspectionDate"`
	EquipmentType      string                 `json:"equipmentType,omitempty"`
	AssignedTo         string                 `json:"assignedTo,omitempty"`
}

func equipmentTypeName(kind *equipmenttype.EquipmentType) string {
	switch {
	case kind == nil:
		return ""
	case kind.Description == "":
		return kind.Code
	default:
		return kind.Code + " — " + kind.Description
	}
}

func equipmentFields() []filtercatalog.Field {
	return []filtercatalog.Field{
		{
			Name:   "status",
			Kind:   filtercatalog.KindEnum,
			Values: equipmentStatuses,
			Note:   noteEquipmentStatus,
		},
		{Name: "ownershipType", Kind: filtercatalog.KindEnum, Values: ownershipTypes},
		{Name: "code", Kind: filtercatalog.KindText, Sortable: true},
		{Name: "make", Kind: filtercatalog.KindText},
		{Name: "model", Kind: filtercatalog.KindText},
		{Name: "licensePlateNumber", Kind: filtercatalog.KindText},
		{Name: "year", Kind: filtercatalog.KindNumber, Sortable: true},
		{
			Name:     "registrationExpiry",
			Kind:     filtercatalog.KindDate,
			Sortable: true,
			Note:     notePlateLapses,
		},
		{Name: "leaseEndDate", Kind: filtercatalog.KindDate, Sortable: true},
		{Name: "createdAt", Kind: filtercatalog.KindDate, Sortable: true},
	}
}

func NewListTractorsTool(
	repo tractorports.TractorRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return listtool.New(listtool.Spec{
		Name:         querytool.ToolListTractors,
		Row:          equipmentRow{},
		EntityPlural: pluralTractors,
		Summary:      tractorsSummary,
		SearchTerms: []string{
			"find tractor by unit number", "locate tractor", "tractor unit number",
			"available tractors",
		},
		Resource:      permission.ResourceTractor,
		Config:        listquery.Describe((*tractor.Tractor)(nil), tractordb.TractorsSchema()),
		Fields:        equipmentFields(),
		Access:        fieldgate.NewAccess(permissions),
		PagesByCursor: true,
		FetchJoined: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			cursor pagination.CursorInfo,
			mayRead func(permission.Resource) bool,
		) (listtool.Page, error) {
			nameDrivers := mayRead(permission.ResourceWorker)
			result, err := repo.List(ctx, &tractorports.ListTractorsRequest{
				Filter: opts,
				Cursor: cursor,
				TractorRelationIncludes: tractorports.TractorRelationIncludes{
					IncludePrimaryWorker: nameDrivers,
					IncludeEquipmentType: true,
				},
			})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.CursorPage(result, func(item *tractor.Tractor) any {
				row := equipmentRow{
					ID:            item.ID.String(),
					Code:          item.Code,
					Status:        string(item.Status),
					Make:          item.Make,
					Model:         item.Model,
					LicensePlate:  item.LicensePlateNumber,
					OwnershipType: string(item.OwnershipType),
				}
				if item.Year != nil {
					row.Year = *item.Year
				}
				row.RegistrationExpiry = querytool.PointerDate(item.RegistrationExpiry)
				row.EquipmentType = equipmentTypeName(item.EquipmentType)
				if nameDrivers {
					row.AssignedTo = querytool.WorkerName(item.PrimaryWorker)
				}

				return row
			})
		},
	})
}

func NewListTrailersTool(repo trailerports.TrailerRepository) agenttoolports.AgentQueryTool {
	fields := append(equipmentFields(), filtercatalog.Field{
		Name:     "lastInspectionDate",
		Kind:     filtercatalog.KindDate,
		Sortable: true,
		Note:     noteLastInspection,
	})

	return listtool.New(listtool.Spec{
		Name:         toolListTrailers,
		Row:          equipmentRow{},
		EntityPlural: pluralTrailers,
		Summary:      trailersSummary,
		SearchTerms: []string{
			"find trailer by unit number", "available dry vans",
		},
		Resource:      permission.ResourceTrailer,
		Config:        listquery.Describe((*trailer.Trailer)(nil), trailerdb.TrailersSchema()),
		Fields:        fields,
		PagesByCursor: true,
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, cursor pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &trailerports.ListTrailersRequest{
				Filter: opts,
				Cursor: cursor,
				TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
					IncludeEquipmentType: true,
				},
			})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.CursorPage(result, func(item *trailer.Trailer) any {
				row := equipmentRow{
					ID:            item.ID.String(),
					Code:          item.Code,
					Status:        string(item.Status),
					Make:          item.Make,
					Model:         item.Model,
					LicensePlate:  item.LicensePlateNumber,
					OwnershipType: string(item.OwnershipType),
				}
				if item.Year != nil {
					row.Year = *item.Year
				}
				row.RegistrationExpiry = querytool.PointerDate(item.RegistrationExpiry)
				row.LastInspectionDate = querytool.PointerDate(item.LastInspectionDate)
				row.EquipmentType = equipmentTypeName(item.EquipmentType)

				return row
			})
		},
	})
}
