package trackingprofile

type Field string

const (
	FieldStatus          = Field("Status")
	FieldLocationNames   = Field("LocationNames")
	FieldWindows         = Field("Windows")
	FieldActuals         = Field("Actuals")
	FieldEta             = Field("Eta")
	FieldReferences      = Field("References")
	FieldCarrier         = Field("Carrier")
	FieldEquipment       = Field("Equipment")
	FieldFreight         = Field("Freight")
	FieldLastLocation    = Field("LastLocation")
	FieldHolds           = Field("Holds")
	FieldNotes           = Field("Notes")
	FieldProofOfDelivery = Field("ProofOfDelivery")
)

func FieldValues() []Field {
	return []Field{
		FieldStatus,
		FieldLocationNames,
		FieldWindows,
		FieldActuals,
		FieldEta,
		FieldReferences,
		FieldCarrier,
		FieldEquipment,
		FieldFreight,
		FieldLastLocation,
		FieldHolds,
		FieldNotes,
		FieldProofOfDelivery,
	}
}

func DefaultFields() []Field {
	return []Field{FieldStatus, FieldWindows, FieldActuals, FieldEta, FieldReferences}
}

func (f Field) IsValid() bool {
	switch f {
	case FieldStatus,
		FieldLocationNames,
		FieldWindows,
		FieldActuals,
		FieldEta,
		FieldReferences,
		FieldCarrier,
		FieldEquipment,
		FieldFreight,
		FieldLastLocation,
		FieldHolds,
		FieldNotes,
		FieldProofOfDelivery:
		return true
	default:
		return false
	}
}
