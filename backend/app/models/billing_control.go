package models

import (
	"errors"
	"github.com/google/uuid"
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

func (bc *BillingControl) BeforeCreate() error {
	return bc.validateBillingControl()
}

func (bc *BillingControl) BeforeUpdate() error {
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
