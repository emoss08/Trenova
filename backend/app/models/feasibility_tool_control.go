package models

import "github.com/google/uuid"

type OperatorChoices string

const (
	OperatorEquals             OperatorChoices = "EQ"
	OperatorNotEquals          OperatorChoices = "NE"
	OperatorGreaterThan        OperatorChoices = "GT"
	OperatorGreaterThanOrEqual OperatorChoices = "GTE"
	OperatorLessThan           OperatorChoices = "LT"
	OperatorLessThanOrEqual    OperatorChoices = "LTE"
)

type FeasibilityToolControl struct {
	TimeStampedModel
	BusinessUnitID uuid.UUID       `gorm:"type:uuid;not null;index"                                         json:"businessUnitId"`
	OrganizationID uuid.UUID       `gorm:"type:uuid;not null;unique"                                        json:"organizationId"`
	OtpOperator    OperatorChoices `gorm:"type:varchar(3);not null;default:'EQ'"                            json:"otpOperator" validate:"required,oneof=EQ NE GT LT LTE"`
	MpwOperator    OperatorChoices `gorm:"type:varchar(3);not null;default:'EQ'"                            json:"mpwOperator" validate:"required,oneof=EQ NE GT LT LTE"`
	MpdOperator    OperatorChoices `gorm:"type:varchar(3);not null;default:'EQ'"                            json:"mpdOperator" validate:"required,oneof=EQ NE GT LT LTE"`
	MpgOperator    OperatorChoices `gorm:"type:varchar(3);not null;default:'EQ'"                            json:"mpgOperator" validate:"required,oneof=EQ NE GT LT LTE"`
	MpwCriteria    float64         `gorm:"type:decimal(10,2);check:mpw_criteria >= 0;not null;default:100"  json:"mpwCriteria" validate:"required,gt=0"`
	MpdCriteria    float64         `gorm:"type:decimal(10,2);check:mpd_criteria >= 0;not null;default:100"  json:"mpdCriteria" validate:"required,gt=0"`
	MpgCriteria    float64         `gorm:"type:decimal(10,2);check:mpg_criteria >= 0;not null;default:100"  json:"mpgCriteria" validate:"required,gt=0"`
	OtpCriteria    float64         `gorm:"type:decimal(10,2);check:otp_criteria >= 0;not null;default:100"  json:"otpCriteria" validate:"required,gt=0"`
}
