package ailog

type Model string

const (
	ModelGPT5Nano         = Model("gpt-5-nano")
	ModelGPT5Nano20250807 = Model("gpt-5-nano-2025-08-07")
	ModelGPT5Mini         = Model("gpt-5-mini")
	ModelGPT5Mini20250807 = Model("gpt-5-mini-2025-08-07")
	ModelModerationLatest = Model("omni-moderation-latest")
)

type Operation string

const (
	OperationClassifyLocation            = Operation("ClassifyLocation")
	OperationDocumentIntelligenceRoute   = Operation("DocumentIntelligenceRoute")
	OperationDocumentIntelligenceExtract = Operation("DocumentIntelligenceExtract")
	OperationShipmentImportChat          = Operation("ShipmentImportChat")
)
