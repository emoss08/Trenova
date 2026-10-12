package modeprofile

import "errors"

var (
	ErrInvalidServiceModel   = errors.New("invalid service model")
	ErrInvalidEquipmentClass = errors.New("invalid equipment class")
	ErrInvalidExecutionParty = errors.New("invalid execution party")
	ErrInvalidProfileStatus  = errors.New("invalid profile status")
	ErrInvalidDeviationState = errors.New("invalid deviation state")
)

type ServiceModel string

const (
	ServiceModelTruckload = ServiceModel("Truckload")
	ServiceModelDedicated = ServiceModel("Dedicated")
	ServiceModelLTL       = ServiceModel("LTL")
	ServiceModelExpedite  = ServiceModel("Expedite")
	ServiceModelFinalMile = ServiceModel("FinalMile")
	ServiceModelDrayage   = ServiceModel("Drayage")
)

func (s ServiceModel) String() string { return string(s) }

func (s ServiceModel) IsValid() bool {
	switch s {
	case ServiceModelTruckload,
		ServiceModelDedicated,
		ServiceModelLTL,
		ServiceModelExpedite,
		ServiceModelFinalMile,
		ServiceModelDrayage:
		return true
	default:
		return false
	}
}

func ServiceModelFromString(v string) (ServiceModel, error) {
	sm := ServiceModel(v)
	if !sm.IsValid() {
		return "", ErrInvalidServiceModel
	}
	return sm, nil
}

type EquipmentClass string

const (
	EquipmentClassDryVan        = EquipmentClass("DryVan")
	EquipmentClassRefrigerated  = EquipmentClass("Refrigerated")
	EquipmentClassFlatbed       = EquipmentClass("Flatbed")
	EquipmentClassTank          = EquipmentClass("Tank")
	EquipmentClassDryBulk       = EquipmentClass("DryBulk")
	EquipmentClassContainer     = EquipmentClass("Container")
	EquipmentClassAutoRack      = EquipmentClass("AutoRack")
	EquipmentClassStraightTruck = EquipmentClass("StraightTruck")
)

func (e EquipmentClass) String() string { return string(e) }

func (e EquipmentClass) IsValid() bool {
	switch e {
	case EquipmentClassDryVan,
		EquipmentClassRefrigerated,
		EquipmentClassFlatbed,
		EquipmentClassTank,
		EquipmentClassDryBulk,
		EquipmentClassContainer,
		EquipmentClassAutoRack,
		EquipmentClassStraightTruck:
		return true
	default:
		return false
	}
}

func EquipmentClassFromString(v string) (EquipmentClass, error) {
	ec := EquipmentClass(v)
	if !ec.IsValid() {
		return "", ErrInvalidEquipmentClass
	}
	return ec, nil
}

type ExecutionParty string

const (
	ExecutionPartyCompanyAsset      = ExecutionParty("CompanyAsset")
	ExecutionPartyOwnerOperator     = ExecutionParty("OwnerOperator")
	ExecutionPartyLeasedOn          = ExecutionParty("LeasedOn")
	ExecutionPartyPurchasedCapacity = ExecutionParty("PurchasedCapacity")
)

func (e ExecutionParty) String() string { return string(e) }

func (e ExecutionParty) IsValid() bool {
	switch e {
	case ExecutionPartyCompanyAsset,
		ExecutionPartyOwnerOperator,
		ExecutionPartyLeasedOn,
		ExecutionPartyPurchasedCapacity:
		return true
	default:
		return false
	}
}

func ExecutionPartyFromString(v string) (ExecutionParty, error) {
	ep := ExecutionParty(v)
	if !ep.IsValid() {
		return "", ErrInvalidExecutionParty
	}
	return ep, nil
}

type ProfileStatus string

const (
	ProfileStatusDraft    = ProfileStatus("Draft")
	ProfileStatusActive   = ProfileStatus("Active")
	ProfileStatusArchived = ProfileStatus("Archived")
)

func (s ProfileStatus) String() string { return string(s) }

func (s ProfileStatus) IsValid() bool {
	switch s {
	case ProfileStatusDraft, ProfileStatusActive, ProfileStatusArchived:
		return true
	default:
		return false
	}
}

func ProfileStatusFromString(v string) (ProfileStatus, error) {
	ps := ProfileStatus(v)
	if !ps.IsValid() {
		return "", ErrInvalidProfileStatus
	}
	return ps, nil
}

type DeviationState string

const (
	DeviationStateOpen         = DeviationState("Open")
	DeviationStateAcknowledged = DeviationState("Acknowledged")
	DeviationStateResolved     = DeviationState("Resolved")
)

func (d DeviationState) String() string { return string(d) }

func (d DeviationState) IsValid() bool {
	switch d {
	case DeviationStateOpen, DeviationStateAcknowledged, DeviationStateResolved:
		return true
	default:
		return false
	}
}

func DeviationStateFromString(v string) (DeviationState, error) {
	ds := DeviationState(v)
	if !ds.IsValid() {
		return "", ErrInvalidDeviationState
	}
	return ds, nil
}
