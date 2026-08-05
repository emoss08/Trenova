package report

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ReportDefinitionRevision)(nil)

type ReportDefinitionRevision struct {
	bun.BaseModel `bun:"table:report_definition_revisions,alias:rdr" json:"-"`

	ID             pulid.ID    `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	DefinitionID   pulid.ID    `json:"definitionId"   bun:"definition_id,type:VARCHAR(100),notnull"`
	RevisionNumber int64       `json:"revisionNumber" bun:"revision_number,type:BIGINT,notnull"`
	CatalogVersion string      `json:"catalogVersion" bun:"catalog_version,type:VARCHAR(80),notnull"`
	Definition     *Definition `json:"definition"     bun:"definition,type:JSONB,notnull"`
	CreatedByID    pulid.ID    `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt      int64       `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	ReportDefinition *ReportDefinition `json:"reportDefinition,omitempty" bun:"rel:belongs-to,join:definition_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (rdr *ReportDefinitionRevision) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if rdr.ID.IsNil() {
			rdr.ID = pulid.MustNew("rdr_")
		}
		rdr.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (rdr *ReportDefinitionRevision) GetID() pulid.ID { return rdr.ID }

func (rdr *ReportDefinitionRevision) GetOrganizationID() pulid.ID { return rdr.OrganizationID }

func (rdr *ReportDefinitionRevision) GetBusinessUnitID() pulid.ID { return rdr.BusinessUnitID }

func (rdr *ReportDefinitionRevision) GetTableName() string { return "report_definition_revisions" }

func (r *ReportDefinitionRevision) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.DefinitionID, validation.Required.Error("Report is required")),
		validation.Field(&r.RevisionNumber,
			validation.Required.Error("Revision number is required"),
			validation.Min(int64(1)).Error("Revision number must be at least one"),
		),
		// The catalog version pins what the definition was written against, so a
		// revision without one cannot be replayed after the catalog moves on.
		validation.Field(&r.CatalogVersion,
			validation.Required.Error("Catalog version is required"),
			validation.Length(1, maxCatalogVersionLength).
				Error("Catalog version cannot be longer than 80 characters"),
		),
		validation.Field(&r.Definition, validation.Required.Error("Definition is required")),
		validation.Field(&r.CreatedByID, validation.Required.Error("Created by is required")),
	))
}

const maxCatalogVersionLength = 80
