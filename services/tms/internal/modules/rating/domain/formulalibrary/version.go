package formulalibrary

import (
	"context"

	"github.com/emoss08/keel"
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
	keel.Table    `                                                          keel:"formula_library_versions"`

	ID             pulid.ID                         `json:"id"             bun:"id,pk,type:VARCHAR(100)"                                                  keel:"id,pk,autoid:flv_"`
	OrganizationID pulid.ID                         `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"                             keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID                         `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"                            keel:"business_unit_id,pk,tenant"`
	LibraryID      pulid.ID                         `json:"libraryId"      bun:"library_id,type:VARCHAR(100),notnull"                                     keel:"library_id"`
	VersionNumber  int64                            `json:"versionNumber"  bun:"version_number,type:BIGINT,notnull"                                       keel:"version_number"`
	Name           string                           `json:"name"           bun:"name,type:VARCHAR(64),notnull"                                            keel:"name"`
	Description    string                           `json:"description"    bun:"description,type:TEXT"                                                    keel:"description,nullzero"`
	Source         string                           `json:"source"         bun:"source,type:TEXT,notnull"                                                 keel:"source"`
	Imports        []tarifftypes.Import             `json:"imports"        bun:"imports,type:JSONB,notnull,default:'[]'"                                  keel:"imports,jsonb"`
	TestCount      int32                            `json:"testCount"      bun:"test_count,type:INTEGER,notnull,default:0"                                keel:"test_count"`
	Status         formulatemplate.Status           `json:"status"         bun:"status,type:formula_template_status_enum,notnull"                         keel:"status"`
	ChangeMessage  string                           `json:"changeMessage"  bun:"change_message,type:TEXT"                                                 keel:"change_message,nullzero"`
	ChangeSummary  map[string]jsonutils.FieldChange `json:"changeSummary"  bun:"change_summary,type:JSONB"                                                keel:"change_summary,jsonb"`
	CreatedByID    pulid.ID                         `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),notnull"                                  keel:"created_by_id"`
	CreatedAt      int64                            `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	CreatedBy *tenant.User `json:"createdBy,omitempty" bun:"rel:belongs-to,join:created_by_id=id" keel:"belongs:created_by_id"`
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
