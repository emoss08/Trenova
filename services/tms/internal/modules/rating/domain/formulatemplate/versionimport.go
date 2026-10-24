package formulatemplate

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type VersionImport struct {
	bun.BaseModel `bun:"table:formula_template_version_imports,alias:ftvi" json:"-"`
	keel.Table    `keel:"formula_template_version_imports"`

	TemplateVersionID    pulid.ID `json:"templateVersionId"    bun:"template_version_id,pk,type:VARCHAR(100)" keel:"template_version_id,pk"`
	LibraryID            pulid.ID `json:"libraryId"            bun:"library_id,pk,type:VARCHAR(100)" keel:"library_id,pk"`
	OrganizationID       pulid.ID `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	TemplateID           pulid.ID `json:"templateId"           bun:"template_id,type:VARCHAR(100),notnull" keel:"template_id"`
	TemplateVersion      int64    `json:"templateVersion"      bun:"template_version,type:BIGINT,notnull" keel:"template_version"`
	LibraryVersionNumber int64    `json:"libraryVersionNumber" bun:"library_version_number,type:BIGINT,notnull" keel:"library_version_number"`
	Path                 string   `json:"path"                 bun:"path,type:VARCHAR(100),notnull" keel:"path"`
}

func (v *FormulaTemplateVersion) ImportRows() []*VersionImport {
	rows := make([]*VersionImport, 0, len(v.Imports))
	seen := make(map[pulid.ID]struct{}, len(v.Imports))
	for i := range v.Imports {
		imp := &v.Imports[i]
		if imp.LibraryID.IsNil() {
			continue
		}
		if _, dup := seen[imp.LibraryID]; dup {
			continue
		}
		seen[imp.LibraryID] = struct{}{}
		rows = append(rows, &VersionImport{
			TemplateVersionID:    v.ID,
			LibraryID:            imp.LibraryID,
			OrganizationID:       v.OrganizationID,
			BusinessUnitID:       v.BusinessUnitID,
			TemplateID:           v.TemplateID,
			TemplateVersion:      v.VersionNumber,
			LibraryVersionNumber: imp.VersionNumber,
			Path:                 imp.Path,
		})
	}
	return rows
}
