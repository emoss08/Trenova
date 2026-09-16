package ailog

// Model is the provider-reported model identifier. It is not a closed set: an
// organization can route work to its own model server, whose models carry
// arbitrary names. The constants below are the ones this system calls by default,
// kept for reference and defaulting rather than for validation.
type Model string

const (
	ModelGPT5Nano         = Model("gpt-5-nano")
	ModelGPT5Nano20250807 = Model("gpt-5-nano-2025-08-07")
	ModelGPT5Mini         = Model("gpt-5-mini")
	ModelGPT5Mini20250807 = Model("gpt-5-mini-2025-08-07")
	ModelModerationLatest = Model("omni-moderation-latest")
	ModelClaudeOpus5      = Model("claude-opus-5")
)

type Operation string

const (
	OperationClassifyLocation            = Operation("ClassifyLocation")
	OperationDocumentIntelligenceRoute   = Operation("DocumentIntelligenceRoute")
	OperationDocumentIntelligenceExtract = Operation("DocumentIntelligenceExtract")
	OperationShipmentImportChat          = Operation("ShipmentImportChat")
	OperationFormulaGenerate             = Operation("FormulaGenerate")
	OperationFormulaExplain              = Operation("FormulaExplain")
)
