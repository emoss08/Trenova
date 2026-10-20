package agentquerytoolservice

// The external tests read through the real GraphQL reader, which this
// package cannot import without a cycle; they reach the constructors here.
var (
	ProvideListEquipmentManufacturersTool = provideListEquipmentManufacturersTool
	NewListEquipmentManufacturersTool     = newListEquipmentManufacturersTool
)
