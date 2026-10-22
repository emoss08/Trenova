package formulalibrary

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*FormulaLibraryVersion)(nil)

type FormulaLibraryVersion struct {
	bun.BaseModel `bun:"table:formula_library_versions,alias:flibv" json:"-"`

	ID             pulid.ID                         `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID                         `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID                         `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	LibraryID      pulid.ID                         `json:"libraryId"      bun:"library_id,type:VARCHAR(100),notnull"`
	VersionNumber  int64                            `json:"versionNumber"  bun:"version_number,type:BIGINT,notnull"`
	Name           string                           `json:"name"           bun:"name,type:VARCHAR(64),notnull"`
	Description    string                           `json:"description"    bun:"description,type:TEXT"`
	Source         string                           `json:"source"         bun:"source,type:TEXT,notnull"`
	Imports        []tarifftypes.Import             `json:"imports"        bun:"imports,type:JSONB,notnull,default:'[]'"`
	TestCount      int32                            `json:"testCount"      bun:"test_count,type:INTEGER,notnull,default:0"`
	Status         formulatemplate.Status           `json:"status"         bun:"status,type:formula_template_status_enum,notnull"`
	ChangeMessage  string                           `json:"changeMessage"  bun:"change_message,type:TEXT"`
	ChangeSummary  map[string]jsonutils.FieldChange `json:"changeSummary"  bun:"change_summary,type:JSONB"`
	CreatedByID    pulid.ID                         `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt      int64                            `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CreatedBy *tenant.User `json:"createdBy,omitempty" bun:"rel:belongs-to,join:created_by_id=id"`
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

func (v *FormulaLibraryVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("flv_")
		}
		if v.Imports == nil {
			v.Imports = []tarifftypes.Import{}
		}
		v.CreatedAt = timeutils.NowUnix()
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
