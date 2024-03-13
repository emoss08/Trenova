package models

// import "github.com/google/uuid"

// type JobFunctionType string

// const (
// 	Manager           JobFunctionType = "MGR"
// 	ManagementTrainee JobFunctionType = "MT"
// 	Supervisor        JobFunctionType = "SP"
// 	Driver            JobFunctionType = "D"
// 	Billing           JobFunctionType = "B"
// 	Finance           JobFunctionType = "F"
// 	Safety            JobFunctionType = "S"
// 	SysAdmin          JobFunctionType = "SA"
// 	Admin             JobFunctionType = "A"
// )

// type JobTitle struct {
// 	BaseModel
// 	OrganizationID uuid.UUID       `gorm:"type:uuid;not null;index" json:"organizationId" validate:"required"`
// 	BusinessUnitID uuid.UUID       `gorm:"type:uuid;not null"       json:"businessUnitId" validate:"required"`
// 	Organization   Organization    `json:"-" validate:"omitempty"`
// 	BusinessUnit   BusinessUnit    `json:"-" validate:"omitempty"`
// 	Status         StatusType      `gorm:"type:status_type;not null;default:'A'" json:"status" validate:"required,len=1,oneof=A I"`
// 	Name           string          `gorm:"type:varchar(100);not null;"           json:"name" validate:"required,max=100"`
// 	Description    *string         `gorm:"type:varchar(100);"                    json:"description" validate:"required,max=100"`
// 	JobFunction    JobFunctionType `gorm:"type:job_function_type;not null;"      json:"jobFunction" validate:"required,len=1,oneof=MGR MT SP D B F S SA A"`
// }
