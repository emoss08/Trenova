package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QualifierCode struct {
	BaseModel
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_qualifier_code_organization_id"                               json:"organizationId" validate:"required"`
	BusinessUnitID uuid.UUID    `gorm:"type:uuid;not null;index"                                                                        json:"businessUnitId" validate:"required"`
	Code           string       `gorm:"type:varchar(10);not null;uniqueIndex:idx_qualifier_code_organization_id,expression:lower(code)" json:"code"           validate:"required,max=10"`
	Description    string       `gorm:"type:varchar(100);not null;"                                                                     json:"description"    validate:"required,max=100"`
	Status         StatusType   `gorm:"type:status_type;not null;default:'A'"                                                           json:"status"         validate:"required,len=1,oneof=A I"`
}

func (qc *QualifierCode) SetOrgID(orgID uuid.UUID) {
	qc.OrganizationID = orgID
}

func (qc *QualifierCode) SetBuID(buID uuid.UUID) {
	qc.BusinessUnitID = buID
}

func (qc *QualifierCode) FetchQualifierCodesForOrg(db *gorm.DB, orgID, buID uuid.UUID, offset, limit int) ([]QualifierCode, int64, error) {
	var qualifierCodes []QualifierCode

	var totalRows int64

	if err := db.
		Model(&QualifierCode{}).
		Where("organization_id = ? AND business_unit_id = ?", orgID, buID).
		Count(&totalRows).Error; err != nil {
		return qualifierCodes, 0, err
	}

	if err := db.
		Model(&QualifierCode{}).
		Where("organization_id = ? AND business_unit_id = ?", orgID, buID).Offset(offset).
		Limit(limit).
		Order("status desc").
		Find(&qualifierCodes).Error; err != nil {
		return qualifierCodes, 0, err
	}

	return qualifierCodes, totalRows, nil
}

func (qc *QualifierCode) FetchQualifierCodeDetails(db *gorm.DB, orgID, buID uuid.UUID, id string) (QualifierCode, error) {
	var qualifierCode QualifierCode

	if err := db.
		Model(&QualifierCode{}).
		Where("organization_id = ? AND id = ? AND business_unit_id = ?", orgID, id, buID).
		First(&qualifierCode).Error; err != nil {
		return qualifierCode, err
	}

	return qualifierCode, nil
}
