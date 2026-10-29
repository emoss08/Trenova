package edi

type PartnerKind string

const (
	PartnerKindInternal = PartnerKind("Internal")
	PartnerKindExternal = PartnerKind("External")
)

type PartnerRole string

const (
	PartnerRoleCustomer  = PartnerRole("Customer")
	PartnerRoleCarrier   = PartnerRole("Carrier")
	PartnerRoleBroker    = PartnerRole("Broker")
	PartnerRoleVendor    = PartnerRole("Vendor")
	PartnerRoleShipper   = PartnerRole("Shipper")
	PartnerRoleConsignee = PartnerRole("Consignee")
	PartnerRoleBillTo    = PartnerRole("BillTo")
)

type ConnectionMethod string

const (
	ConnectionMethodInternal = ConnectionMethod("Internal")
	ConnectionMethodAS2      = ConnectionMethod("AS2")
	ConnectionMethodSFTP     = ConnectionMethod("SFTP")
	ConnectionMethodVAN      = ConnectionMethod("VAN")
)

type ConnectionStatus string

const (
	ConnectionStatusPendingAcceptance = ConnectionStatus("PendingAcceptance")
	ConnectionStatusActive            = ConnectionStatus("Active")
	ConnectionStatusSuspended         = ConnectionStatus("Suspended")
	ConnectionStatusRejected          = ConnectionStatus("Rejected")
	ConnectionStatusRevoked           = ConnectionStatus("Revoked")
)

type MappingEntityType string

const (
	MappingEntityTypeCustomer                 = MappingEntityType("Customer")
	MappingEntityTypeServiceType              = MappingEntityType("ServiceType")
	MappingEntityTypeShipmentType             = MappingEntityType("ShipmentType")
	MappingEntityTypeFormulaTemplate          = MappingEntityType("FormulaTemplate")
	MappingEntityTypeLocation                 = MappingEntityType("Location")
	MappingEntityTypeCommodity                = MappingEntityType("Commodity")
	MappingEntityTypeAccessorialCharge        = MappingEntityType("AccessorialCharge")
	MappingEntityTypeServiceFailureReasonCode = MappingEntityType("ServiceFailureReasonCode")
)

func (k PartnerKind) IsValid() bool {
	switch k {
	case PartnerKindInternal,
		PartnerKindExternal:
		return true
	default:
		return false
	}
}

func (r PartnerRole) IsValid() bool {
	switch r {
	case PartnerRoleCustomer,
		PartnerRoleCarrier,
		PartnerRoleBroker,
		PartnerRoleVendor,
		PartnerRoleShipper,
		PartnerRoleConsignee,
		PartnerRoleBillTo:
		return true
	default:
		return false
	}
}

func (m ConnectionMethod) IsValid() bool {
	switch m {
	case ConnectionMethodInternal,
		ConnectionMethodAS2,
		ConnectionMethodSFTP,
		ConnectionMethodVAN:
		return true
	default:
		return false
	}
}

func (s ConnectionStatus) IsValid() bool {
	switch s {
	case ConnectionStatusPendingAcceptance,
		ConnectionStatusActive,
		ConnectionStatusSuspended,
		ConnectionStatusRejected,
		ConnectionStatusRevoked:
		return true
	default:
		return false
	}
}

func (t MappingEntityType) IsValid() bool {
	switch t {
	case MappingEntityTypeCustomer,
		MappingEntityTypeServiceType,
		MappingEntityTypeShipmentType,
		MappingEntityTypeFormulaTemplate,
		MappingEntityTypeLocation,
		MappingEntityTypeCommodity,
		MappingEntityTypeAccessorialCharge,
		MappingEntityTypeServiceFailureReasonCode:
		return true
	default:
		return false
	}
}
