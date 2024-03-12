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
	Kafka    SourceType = "KAFKA"
	DataBase SourceType = "DB"
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
	OrganizationID   uuid.UUID          `gorm:"type:uuid;not null;"                 json:"organizationId"   validate:"required"`
	BusinessUnitID   uuid.UUID          `gorm:"type:uuid;not null"                  json:"businessUnitId"   validate:"required"`
	BusinessUnit     BusinessUnit       `json:"-"`
	Status           StatusType         `gorm:"type:status_type;not null;default:'A'" json:"status"         validate:"required,len=1,oneof=A I"`
	Name             string             `gorm:"type:varchar(50);not null;"         json:"name"              validate:"required,max=50"`
	DatabaseAction   DatabaseActionType `gorm:"type:database_action_type;not null" json:"databaseAction"    validate:"required,max=6,oneof=INSERT UPDATE DELETE"`
	Source           SourceType         `gorm:"type:table_change_type;not null"    json:"source"            validate:"required,max=6,oneof=KAFKA DB"`
	EmailProfile     EmailProfile       `gorm:"foreignKey:EmailProfileID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	TableName        *string            `gorm:"type:varchar(255);"                 json:"tableName"         validate:"max=255,required_if=Source DB"`
	Topic            *string            `gorm:"type:varchar(255)"                  json:"topic"             validate:"max=255,required_if=Source KAFKA"`
	Description      *string            `gorm:"type:text"                          json:"description"       validate:"omitempty"`
	CustomSubject    *string            `gorm:"type:varchar(255)"                  json:"customSubject"     validate:"omitempty,max=255"`
	FunctionName     *string            `gorm:"type:varchar(50)"                   json:"functionName"      validate:"omitempty,max=50"`
	TriggerName      *string            `gorm:"type:varchar(50)"                   json:"triggerName"       validate:"omitempty,max=50"`
	ListenerName     *string            `gorm:"type:varchar(50)"                   json:"listenerName"      validate:"omitempty,max=50"`
	EmailRecipients  *string            `gorm:"type:text"                          json:"emailRecipients"   validate:"omitempty,commaSeparatedEmails"`
	EmailProfileID   *uuid.UUID         `gorm:"type:uuid"                          json:"emailProfileId"    validate:"omitempty"`
	ConditionalLogic *datatypes.JSON    `gorm:"type:json"                          json:"conditionalLogic"  validate:"omitempty"`
	EffectiveDate    *time.Time         `gorm:"type:date"                          json:"effectiveDate"     validate:"omitempty"`
	ExpirationDate   *time.Time         `gorm:"type:date"                          json:"expirationDate"    validate:"omitempty"`
}

var (
	errTopicRequiredKafka = errors.New("topic is required when the source is KAFKA")

	errTableNameRequiredDB = errors.New("table name is required when the source is DB")

	errDeleteActionOnlyForDB = errors.New("DELETE action is only valid for DB source")

	errEffectiveDateBeforeExpirationDate = errors.New("effective date must be before expiration date")
)

func (tbc *TableChangeAlert) validateTableChangeAlert() error {
	if tbc.Source == Kafka && tbc.Topic == nil {
		return errTopicRequiredKafka
	}

	if tbc.Source == DataBase && tbc.TableName == nil {
		return errTableNameRequiredDB
	}

	if tbc.DatabaseAction == Delete && tbc.Source != DataBase {
		return errDeleteActionOnlyForDB
	}

	if tbc.EffectiveDate != nil && tbc.ExpirationDate != nil && tbc.EffectiveDate.After(*tbc.ExpirationDate) {
		return errEffectiveDateBeforeExpirationDate
	}

	if tbc.Source == Kafka {
		tbc.TableName = nil
	}

	if tbc.Source == DataBase {
		tbc.Topic = nil
	}

	return nil
}

func (tbc *TableChangeAlert) BeforeCreate(_ *gorm.DB) error {
	return tbc.validateTableChangeAlert()
}

func (tbc *TableChangeAlert) BeforeUpdate(_ *gorm.DB) error {
	return tbc.validateTableChangeAlert()
}
