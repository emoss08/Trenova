package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuditLog struct {
	OrganizationID uuid.UUID      `gorm:"type:uuid;index;" json:"organizationId" validate:"required,uuid"`
	Organization   Organization   `gorm:"foreignkey:OrganizationID;" json:"organization"`
	BusinessUnitID uuid.UUID      `gorm:"type:uuid;" json:"businessUnitId" validate:"required,uuid"`
	BusinessUnit   BusinessUnit   `gorm:"foreignkey:BusinessUnitID;" json:"businessUnit"`
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;" json:"id"`
	Action         string         `gorm:"size:255;" json:"action" validate:"required"`
	UserID         uuid.UUID      `gorm:"type:uuid;" json:"userId" validate:"required"`
	User           User           `gorm:"foreignkey:UserID;" json:"user"`
	RecordID       uuid.UUID      `gorm:"type:uuid;" json:"recordId" validate:"required"`
	Data           datatypes.JSON `gorm:"type:jsonb;" json:"data" validate:"required,json"`
	CreatedAt      time.Time      `gorm:"default:CURRENT_TIMESTAMP;index;" json:"createdAt"`
}

func CreateAuditLog(tx *gorm.DB, action string, user User, recordID uuid.UUID, data datatypes.JSON) error {
	auditLog := AuditLog{
		Action:         action,
		UserID:         user.ID,
		RecordID:       recordID,
		Data:           data,
		OrganizationID: user.OrganizationID, // Assuming the user has an OrganizationID field
		BusinessUnitID: user.BusinessUnitID, // Assuming this is relevant
	}

	return tx.Create(&auditLog).Error
}

func GetAuditLogsForOrg(tx *gorm.DB, orgID uuid.UUID) ([]AuditLog, error) {
	var auditLogs []AuditLog

	err := tx.Where("organization_id = ?", orgID).Find(&auditLogs).Error

	return auditLogs, err
}
