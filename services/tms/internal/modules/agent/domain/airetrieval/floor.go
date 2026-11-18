package airetrieval

const (
	DefaultSimilarityFloor = 0.5
	TunedEmbeddingModel    = "nomic-embed-text"
)

var modelSimilarityFloors = map[string]float64{
	TunedEmbeddingModel: 0.58,
}

func SimilarityFloor(modelKey string) float64 {
	if floor, ok := modelSimilarityFloors[ModelKeyModel(modelKey)]; ok {
		return floor
	}

	return DefaultSimilarityFloor
}

func IsTunedModel(modelKey string) bool {
	return modelKey != "" && ModelKeyModel(modelKey) == TunedEmbeddingModel
}
