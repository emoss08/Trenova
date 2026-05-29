package distancemileagejobs

import "github.com/emoss08/trenova/internal/core/domain/storedmileage"

type FlushStoredMileageBufferResult struct {
	RecordCount int                              `json:"recordCount"`
	Batches     [][]*storedmileage.StoredMileage `json:"batches"`
}

type UpsertStoredMileageBatchPayload struct {
	Records []*storedmileage.StoredMileage `json:"records"`
}

type UpsertStoredMileageBatchResult struct {
	ProcessedCount int `json:"processedCount"`
}
