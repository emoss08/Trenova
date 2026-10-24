package formulatemplate

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type VersionTable struct {
	bun.BaseModel `bun:"table:formula_template_version_tables,alias:ftvt" json:"-"`
	keel.Table    `keel:"formula_template_version_tables"`

	TemplateVersionID   pulid.ID `json:"templateVersionId"   bun:"template_version_id,pk,type:VARCHAR(100)" keel:"template_version_id,pk"`
	RateMatrixID        pulid.ID `json:"rateMatrixId"        bun:"rate_matrix_id,pk,type:VARCHAR(100)" keel:"rate_matrix_id,pk"`
	OrganizationID      pulid.ID `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	TemplateID          pulid.ID `json:"templateId"          bun:"template_id,type:VARCHAR(100),notnull" keel:"template_id"`
	TemplateVersion     int64    `json:"templateVersion"     bun:"template_version,type:BIGINT,notnull" keel:"template_version"`
	MatrixVersionNumber int64    `json:"matrixVersionNumber" bun:"matrix_version_number,type:BIGINT,notnull" keel:"matrix_version_number"`
	Code                string   `json:"code"                bun:"code,type:VARCHAR(64),notnull" keel:"code"`
}

func (v *FormulaTemplateVersion) TableRows() []*VersionTable {
	rows := make([]*VersionTable, 0, len(v.Tables))
	seen := make(map[pulid.ID]struct{}, len(v.Tables))
	for i := range v.Tables {
		pin := &v.Tables[i]
		if pin.RateMatrixID.IsNil() || pin.VersionNumber <= 0 {
			continue
		}
		if _, dup := seen[pin.RateMatrixID]; dup {
			continue
		}
		seen[pin.RateMatrixID] = struct{}{}
		rows = append(rows, &VersionTable{
			TemplateVersionID:   v.ID,
			RateMatrixID:        pin.RateMatrixID,
			OrganizationID:      v.OrganizationID,
			BusinessUnitID:      v.BusinessUnitID,
			TemplateID:          v.TemplateID,
			TemplateVersion:     v.VersionNumber,
			MatrixVersionNumber: pin.VersionNumber,
			Code:                pin.Code,
		})
	}
	return rows
}
