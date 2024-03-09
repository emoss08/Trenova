package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SourceType string

const (
	Kafka SourceType = "KAFKA"
	Db    SourceType = "DB"
)

type DatabaseActionType string

const (
	Insert DatabaseActionType = "INSERT"
	Update DatabaseActionType = "UPDATE"
	Delete DatabaseActionType = "DELETE"
	All    DatabaseActionType = "ALL"
)

type TableChangeAlert struct {
	TimeStampedModel
	OrganizationID   uuid.UUID          `gorm:"type:uuid;not null;" json:"organizationId"`
	Organization     Organization       `json:"-"`
	BusinessUnitID   uuid.UUID          `gorm:"type:uuid;not null" json:"businessUnitId"`
	BusinessUnit     BusinessUnit       `json:"-"`
	Status           StatusType         `json:"status" gorm:"type:status_type;not null;default:'A'" validate:"required,len=1,oneof=A I"`
	Name             string             `json:"name" gorm:"type:varchar(50);not null;" validate:"required,max=50"`
	DatabaseAction   DatabaseActionType `json:"databaseAction" gorm:"type:database_action_type;not null" validate:"required,max=6,oneof=INSERT UPDATE DELETE"`
	Source           SourceType         `json:"source" gorm:"type:table_change_type;not null" validate:"required,max=6,oneof=KAFKA DB"`
	TableName        *string            `json:"tableName" gorm:"type:varchar(255);" validate:"required,max=255"`
	Topic            *string            `json:"topic" gorm:"type:varchar(255)" validate:"omitempty,max=255"`
	Description      *string            `json:"description" gorm:"type:text" validate:"omitempty"`
	EmailProfile     EmailProfile       `json:"-" gorm:"foreignKey:EmailProfileID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	EmailProfileID   *uuid.UUID         `json:"emailProfileId" gorm:"type:uuid"`
	EmailRecipients  *string            `json:"emailRecipients" gorm:"type:text" validate:"omitempty,commaSeparatedEmails"`
	ConditionalLogic *datatypes.JSON    `json:"conditionalLogic" gorm:"type:json" validate:"omitempty"`
	CustomSubject    *string            `json:"customSubject" gorm:"type:varchar(255)" validate:"omitempty,max=255"`
	FunctionName     *string            `json:"functionName" gorm:"type:varchar(50)" validate:"omitempty,max=50"`
	TriggerName      *string            `json:"triggerName" gorm:"type:varchar(50)" validate:"omitempty,max=50"`
	ListenerName     *string            `json:"listenerName" gorm:"type:varchar(50)" validate:"omitempty,max=50"`
	EffectiveDate    *time.Time         `json:"effectiveDate" gorm:"type:date" validate:"omitempty"`
	ExpirationDate   *time.Time         `json:"expirationDate" gorm:"type:date" validate:"omitempty"`
}

func (tbc *TableChangeAlert) validateTableChangeAlert() error {
	if tbc.Source == Kafka && tbc.Topic == nil {
		return errors.New("topic is required when the source is KAFKA")
	}
	if tbc.Source == Db && tbc.TableName == nil {
		return errors.New("tableName is required when the source is DB")
	}
	if tbc.DatabaseAction == Delete && tbc.Source != Db {
		return errors.New("DELETE action is only valid for DB source")
	}
	if tbc.EffectiveDate != nil && tbc.ExpirationDate != nil && tbc.EffectiveDate.After(*tbc.ExpirationDate) {
		return errors.New("effective date must be before expiration date")
	}
	if tbc.Source == Kafka {
		tbc.TableName = nil
	}
	if tbc.Source == Db {
		tbc.Topic = nil
	}
	return nil
}

func (tbc *TableChangeAlert) BeforeCreate(tx *gorm.DB) error {
	return tbc.validateTableChangeAlert()
}

func (tbc *TableChangeAlert) BeforeUpdate(tx *gorm.DB) error {
	return tbc.validateTableChangeAlert()
}
