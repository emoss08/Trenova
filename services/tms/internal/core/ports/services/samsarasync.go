package services

type SamsaraWorkerSyncFailure struct {
	WorkerID  string `json:"workerId"`
	Worker    string `json:"worker"`
	Operation string `json:"operation"`
	Message   string `json:"message"`
}

type SamsaraWorkerSyncResult struct {
	TotalWorkers          int                        `json:"totalWorkers"`
	ActiveWorkers         int                        `json:"activeWorkers"`
	RemoteDrivers         int                        `json:"remoteDrivers"`
	AlreadyMapped         int                        `json:"alreadyMapped"`
	MappedFromExternalIDs int                        `json:"mappedFromExternalIds"`
	CreatedDrivers        int                        `json:"createdDrivers"`
	UpdatedRemoteDrivers  int                        `json:"updatedRemoteDrivers"`
	UpdatedMappings       int                        `json:"updatedMappings"`
	SkippedInactive       int                        `json:"skippedInactive"`
	Failed                int                        `json:"failed"`
	Failures              []SamsaraWorkerSyncFailure `json:"failures,omitempty"`
}

type WorkerSyncReadinessResponse struct {
	TotalWorkers           int   `json:"totalWorkers"`
	ActiveWorkers          int   `json:"activeWorkers"`
	SyncedActiveWorkers    int   `json:"syncedActiveWorkers"`
	UnsyncedActiveWorkers  int   `json:"unsyncedActiveWorkers"`
	AllActiveWorkersSynced bool  `json:"allActiveWorkersSynced"`
	LastCalculatedAt       int64 `json:"lastCalculatedAt"`
}

type WorkerSyncDrift struct {
	WorkerID        string `json:"workerId"`
	WorkerName      string `json:"workerName"`
	DriftType       string `json:"driftType"`
	Message         string `json:"message"`
	LocalExternalID string `json:"localExternalId,omitempty"`
	RemoteDriverID  string `json:"remoteDriverId,omitempty"`
	DetectedAt      int64  `json:"detectedAt"`
}

type WorkerSyncDriftResponse struct {
	Drifts              []WorkerSyncDrift `json:"drifts"`
	TotalDrifts         int               `json:"totalDrifts"`
	WorkersWithDrift    int               `json:"workersWithDrift"`
	MissingMapping      int               `json:"missingMapping"`
	MissingRemoteDriver int               `json:"missingRemoteDriver"`
	MappingMismatch     int               `json:"mappingMismatch"`
	RemoteDeactivated   int               `json:"remoteDeactivated"`
	LastCalculatedAt    int64             `json:"lastCalculatedAt"`
}

type RepairWorkerSyncDriftRequest struct {
	WorkerIDs []string `json:"workerIds"`
}

type RepairWorkerSyncDriftResponse struct {
	RequestedWorkers int                        `json:"requestedWorkers"`
	RepairedWorkers  int                        `json:"repairedWorkers"`
	FailedWorkers    int                        `json:"failedWorkers"`
	Failures         []SamsaraWorkerSyncFailure `json:"failures,omitempty"`
}
