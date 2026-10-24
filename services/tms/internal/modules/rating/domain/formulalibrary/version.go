package formulalibrary

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type FormulaLibraryVersion struct {
	keel.Table `                                                          keel:"formula_library_versions"`

	ID             pulid.ID                         `json:"id" keel:"id,pk,autoid:flv_"`
	OrganizationID pulid.ID                         `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID                         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	LibraryID      pulid.ID                         `json:"libraryId" keel:"library_id"`
	VersionNumber  int64                            `json:"versionNumber" keel:"version_number"`
	Name           string                           `json:"name" keel:"name"`
	Description    string                           `json:"description" keel:"description,nullzero"`
	Source         string                           `json:"source" keel:"source"`
	Imports        []tarifftypes.Import             `json:"imports" keel:"imports,jsonb"`
	TestCount      int32                            `json:"testCount" keel:"test_count"`
	Status         formulatemplate.Status           `json:"status" keel:"status"`
	ChangeMessage  string                           `json:"changeMessage" keel:"change_message,nullzero"`
	ChangeSummary  map[string]jsonutils.FieldChange `json:"changeSummary" keel:"change_summary,jsonb"`
	CreatedByID    pulid.ID                         `json:"createdById" keel:"created_by_id"`
	CreatedAt      int64                            `json:"createdAt" keel:"created_at,autocreate"`

	CreatedBy *tenant.User `json:"createdBy,omitempty" keel:"belongs:created_by_id"`
}

func (v *FormulaLibraryVersion) Path() string {
	return Path(v.Name)
}

func (v *FormulaLibraryVersion) Pin() tarifftypes.Import {
	return tarifftypes.Import{
		Path:          v.Path(),
		LibraryID:     v.LibraryID,
		VersionNumber: v.VersionNumber,
	}
}

func (v *FormulaLibraryVersion) BeforeCreate(_ context.Context) error {
	if v.Imports == nil {
		v.Imports = []tarifftypes.Import{}
	}

	return nil
}

func (v *FormulaLibraryVersion) GetID() pulid.ID {
	return v.ID
}

func (v *FormulaLibraryVersion) GetOrganizationID() pulid.ID {
	return v.OrganizationID
}

func (v *FormulaLibraryVersion) GetBusinessUnitID() pulid.ID {
	return v.BusinessUnitID
}

func (v *FormulaLibraryVersion) GetTableName() string {
	return "formula_library_versions"
}

func NewVersionFromLibrary(
	library *FormulaLibrary,
	versionNumber int64,
	createdByID pulid.ID,
	changeMessage string,
	changeSummary map[string]jsonutils.FieldChange,
) *FormulaLibraryVersion {
	return &FormulaLibraryVersion{
		OrganizationID: library.OrganizationID,
		BusinessUnitID: library.BusinessUnitID,
		LibraryID:      library.ID,
		VersionNumber:  versionNumber,
		Name:           library.Name,
		Description:    library.Description,
		Source:         library.Source,
		Imports:        library.Imports,
		TestCount:      library.TestCount,
		Status:         library.Status,
		ChangeMessage:  changeMessage,
		ChangeSummary:  changeSummary,
		CreatedByID:    createdByID,
	}
}
