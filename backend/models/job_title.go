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
	Status      Status      `gorm:"size:1;"`
	Description *string     `gorm:"type:text;"`
	JobFunction JobFunction `gorm:"size:18;" json:"jobFunction"`
}
