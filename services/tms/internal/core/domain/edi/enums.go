package edi

type PartnerKind string

const (
	PartnerKindInternal PartnerKind = "Internal"
	PartnerKindExternal PartnerKind = "External"
)

type PartnerRole string

const (
	PartnerRoleCustomer  PartnerRole = "Customer"
	PartnerRoleCarrier   PartnerRole = "Carrier"
	PartnerRoleBroker    PartnerRole = "Broker"
	PartnerRoleVendor    PartnerRole = "Vendor"
	PartnerRoleShipper   PartnerRole = "Shipper"
	PartnerRoleConsignee PartnerRole = "Consignee"
	PartnerRoleBillTo    PartnerRole = "BillTo"
)

type MappingEntityType string

const (
	MappingEntityTypeCustomer          MappingEntityType = "Customer"
	MappingEntityTypeServiceType       MappingEntityType = "ServiceType"
	MappingEntityTypeShipmentType      MappingEntityType = "ShipmentType"
	MappingEntityTypeFormulaTemplate   MappingEntityType = "FormulaTemplate"
	MappingEntityTypeLocation          MappingEntityType = "Location"
	MappingEntityTypeCommodity         MappingEntityType = "Commodity"
	MappingEntityTypeAccessorialCharge MappingEntityType = "AccessorialCharge"
)

type TransferStatus string

const (
	TransferStatusSubmitted       TransferStatus = "Submitted"
	TransferStatusMappingRequired TransferStatus = "MappingRequired"
	TransferStatusPendingApproval TransferStatus = "PendingApproval"
	TransferStatusProcessing      TransferStatus = "Processing"
	TransferStatusApproved        TransferStatus = "Approved"
	TransferStatusRejected        TransferStatus = "Rejected"
	TransferStatusCanceled        TransferStatus = "Canceled"
	TransferStatusFailed          TransferStatus = "Failed"
)

func (s TransferStatus) IsFinal() bool {
	switch s {
	case TransferStatusApproved,
		TransferStatusRejected,
		TransferStatusCanceled,
		TransferStatusFailed:
		return true
	default:
		return false
	}
}

func (s TransferStatus) IsActionable() bool {
	return !s.IsFinal() && s != TransferStatusProcessing
}
