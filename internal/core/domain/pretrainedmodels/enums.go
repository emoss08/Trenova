package pretrainedmodels

type ModelType string

const (
	// ModelTypeDocumentQuality is a model that assesses the quality of a document
	ModelTypeDocumentQuality = ModelType("DocumentQuality")
)

type ModelStatus string

const (
	// ModelStatusStable is a model that is stable and ready for production
	ModelStatusStable = ModelStatus("Stable")

	// ModelStatusBeta is a model that is in beta testing
	ModelStatusBeta = ModelStatus("Beta")

	// ModelStatusLegacy is a model that is deprecated and no longer in use
	ModelStatusLegacy = ModelStatus("Legacy")
)
