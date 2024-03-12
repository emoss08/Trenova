package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AutoBillShipmentType string

const (
	// AutoBillShipmentDelivery is a constant for the AutoBillShipmentType enum. When a shipment is delivered.
	AutoBillShipmentDelivery AutoBillShipmentType = "D"

	// AutoBillShipmentTransferred is a constant for the AutoBillShipmentType enum. When a shipment is transferred to billing queue.
	AutoBillShipmentTransferred AutoBillShipmentType = "T"

	// AutoBillingMarkedReady is a constant for the AutoBillShipmentType enum. When a shipment is marked ready to bill.
	AutoBillingMarkedReady AutoBillShipmentType = "MR"
)

type ShipmentTransferCriteriaType string

const (
	// ShipmentTransferCriteriaRAndC  is a constant for the ShipmentTransferCriteriaType enum. When a shipment is ready to bill and confirmed.
	ShipmentTransferCriteriaRAndC ShipmentTransferCriteriaType = "RC"

	// ShipmentTransferCriteriaCompleted is a constant for the ShipmentTransferCriteriaType enum. When a shipment is completed.
	ShipmentTransferCriteriaCompleted ShipmentTransferCriteriaType = "C"

	// ShipmentTransferCriteriaReady is a constant for the ShipmentTransferCriteriaType enum. When a shipment is ready to bill.
	ShipmentTransferCriteriaReady ShipmentTransferCriteriaType = "RTB"
)

type BillingControl struct {
	TimeStampedModel
	BusinessUnitID           uuid.UUID                    `gorm:"type:uuid;not null;index"                          json:"businessUnitId"`
	OrganizationID           uuid.UUID                    `gorm:"type:uuid;not null;unique"                         json:"organizationId"`
	RemoveBillingHistory     bool                         `gorm:"type:boolean;not null;default:false"               json:"removeBillingHistory"     validate:"omitempty"`
	AutoBillShipment         bool                         `gorm:"type:boolean;not null;default:false"               json:"autoBillShipment"         validate:"omitempty"`
	AutoMarkReadyToBill      bool                         `gorm:"type:boolean;not null;default:false"               json:"autoMarkReadyToBill"      validate:"omitempty"`
	ValidateCustomerRates    bool                         `gorm:"type:boolean;not null;default:false"               json:"validateCustomerRates"    validate:"omitempty"`
	EnforceCustomerBilling   bool                         `gorm:"type:boolean;not null;default:false"               json:"enforceCustomerBilling"   validate:"omitempty"`
	AutoBillCriteria         AutoBillShipmentType         `gorm:"type:auto_billing_shipment_type;default:'MR'"      json:"autoBillCriteria"         validate:"omitempty,oneof=D T MR"`
	ShipmentTransferCriteria ShipmentTransferCriteriaType `gorm:"type:shipment_transfer_criteria_type;default:'RC'" json:"shipmentTransferCriteria" validate:"omitempty,oneof=RC C RTB"`
}

func (bc *BillingControl) BeforeCreate(_ *gorm.DB) error {
	return bc.validateBillingControl()
}

func (bc *BillingControl) BeforeUpdate(_ *gorm.DB) error {
	return bc.validateBillingControl()
}

var errAutoBillCriteria = errors.New("AutoBillCriteria must be set if AutoBillShipment is true")

func (bc *BillingControl) validateBillingControl() error {
	// if autoBillShipment is true and not AutoBillCriteria is set, return an error
	if bc.AutoBillShipment && bc.AutoBillCriteria == "" {
		return errAutoBillCriteria
	}

	return nil
}
