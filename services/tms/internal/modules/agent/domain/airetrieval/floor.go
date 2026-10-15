package airetrieval

const DefaultSimilarityFloor = 0.5

var modelSimilarityFloors = map[string]float64{
	"nomic-embed-text": 0.58,
}

func SimilarityFloor(modelKey string) float64 {
	if floor, ok := modelSimilarityFloors[ModelKeyModel(modelKey)]; ok {
		return floor
	}

	return DefaultSimilarityFloor
}
