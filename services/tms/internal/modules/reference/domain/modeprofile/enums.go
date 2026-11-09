package modeprofile

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
