package formulatemplate

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type VersionTable struct {
	keel.Table `keel:"formula_template_version_tables"`

	TemplateVersionID   pulid.ID `json:"templateVersionId"   keel:"template_version_id,pk"`
	RateMatrixID        pulid.ID `json:"rateMatrixId"        keel:"rate_matrix_id,pk"`
	OrganizationID      pulid.ID `json:"organizationId"      keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	TemplateID          pulid.ID `json:"templateId"          keel:"template_id"`
	TemplateVersion     int64    `json:"templateVersion"     keel:"template_version"`
	MatrixVersionNumber int64    `json:"matrixVersionNumber" keel:"matrix_version_number"`
	Code                string   `json:"code"                keel:"code"`
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
