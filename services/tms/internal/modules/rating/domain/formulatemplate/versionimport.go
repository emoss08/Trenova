package formulatemplate

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type VersionImport struct {
	keel.Table `keel:"formula_template_version_imports"`

	TemplateVersionID    pulid.ID `json:"templateVersionId" keel:"template_version_id,pk"`
	LibraryID            pulid.ID `json:"libraryId" keel:"library_id,pk"`
	OrganizationID       pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	TemplateID           pulid.ID `json:"templateId" keel:"template_id"`
	TemplateVersion      int64    `json:"templateVersion" keel:"template_version"`
	LibraryVersionNumber int64    `json:"libraryVersionNumber" keel:"library_version_number"`
	Path                 string   `json:"path" keel:"path"`
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
