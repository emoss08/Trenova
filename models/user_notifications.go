package models

// import "github.com/google/uuid"

// type UserNotifications struct {
// 	BaseModel
// 	Organization   Organization `json:"-" validate:"omitempty"`
// 	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
// 	User           User         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-" `
// 	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index"                                        json:"organizationId" validate:"required"`
// 	BusinessUnitID uuid.UUID    `gorm:"type:uuid;not null"                                              json:"businessUnitId" validate:"required"`
// 	UserID         uuid.UUID    `gorm:"type:uuid;not null"                                              json:"userId"         validate:"required"`
// 	Verb           string       `gorm:"type:varchar(255);not null"                                      json:"verb"           validate:"required,max=255"`
// 	Description    string       `gorm:"type:text;not null"                                              json:"description"    validate:"required"`
// }
