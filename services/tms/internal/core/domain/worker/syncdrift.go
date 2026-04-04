package worker

import "github.com/uptrace/bun"

type WorkerSyncDrift struct {
	bun.BaseModel `bun:"table:samsara_worker_sync_drifts,alias:wsd"`

	OrganizationID  string `bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  string `bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	WorkerID        string `bun:"worker_id,pk,type:VARCHAR(100),notnull"`
	DriftType       string `bun:"drift_type,pk,type:VARCHAR(64),notnull"`
	WorkerName      string `bun:"worker_name,type:VARCHAR(255),notnull"`
	Message         string `bun:"message,type:TEXT,notnull"`
	LocalExternalID string `bun:"local_external_id,type:TEXT,nullzero"`
	RemoteDriverID  string `bun:"remote_driver_id,type:TEXT,nullzero"`
	DetectedAt      int64  `bun:"detected_at,type:BIGINT,notnull"`
}
