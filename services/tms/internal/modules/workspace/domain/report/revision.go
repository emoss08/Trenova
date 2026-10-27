package report

import (
	"github.com/emoss08/keel/analytics/ir"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type ReportDefinitionRevision struct {
	keel.Table `keel:"report_definition_revisions"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:rdr_"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	DefinitionID   pulid.ID       `json:"definitionId"   keel:"definition_id"`
	RevisionNumber int64          `json:"revisionNumber" keel:"revision_number"`
	CatalogVersion string         `json:"catalogVersion" keel:"catalog_version"`
	Definition     *ir.Definition `json:"definition"     keel:"definition,jsonb"`
	CreatedByID    pulid.ID       `json:"createdById"    keel:"created_by_id"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at,autocreate"`

	ReportDefinition *ReportDefinition `json:"reportDefinition,omitempty" keel:"belongs:definition_id"`
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
