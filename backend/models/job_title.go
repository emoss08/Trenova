package models

type Status string

const (
	ACTIVE   Status = "A"
	INACTIVE Status = "I"
)

type JobFunction string

const (
	MANAGER    JobFunction = "MANAGER"
	MT         JobFunction = "MANAGEMENT_TRAINEE"
	SUPERVISOR JobFunction = "SUPERVISOR"
	DISPATCHER JobFunction = "DISPATCHER"
	BILLING    JobFunction = "BILLING"
	FINANCE    JobFunction = "FINANCE"
	SAFEY      JobFunction = "SAFETY"
	DRIVER     JobFunction = "DRIVER"
	MECHANIC   JobFunction = "MECHANIC"
	SYS_ADMIN  JobFunction = "SYS_ADMIN"
)

type JobTitle struct {
	BaseModel
	Status      Status      `gorm:"size:1;type:status_type" json:"status" validate:"requried,oneof=A I,max=1"`
	Name        string      `gorm:"size:100;" json:"name" validate:"required,max=100"`
	Description *string     `gorm:"type:text;" json:"description" validate:"omitempty"`
	JobFunction JobFunction `gorm:"size:18;type:job_function_type" json:"jobFunction" validate:"required,oneof=MANAGER MANAGEMENT_TRAINEE SUPERVISOR DISPATCHER BILLING FINANCE SAFETY DRIVER MECHANIC SYS_ADMIN,max=18"`
}
