package reportjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

type RunReportPayload struct {
	RunID          pulid.ID `json:"runId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type PreparedRun struct {
	RunID          pulid.ID       `json:"runId"`
	OrganizationID pulid.ID       `json:"organizationId"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	RequestedByID  pulid.ID       `json:"requestedById"`
	RevisionID     pulid.ID       `json:"revisionId"`
	ScheduleID     pulid.ID       `json:"scheduleId,omitempty"`
	CannedKey      string         `json:"cannedKey,omitempty"`
	Format         report.Format  `json:"format"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Params         map[string]any `json:"params"`
	OrgTimezone    string         `json:"orgTimezone"`
	RequestedBy    string         `json:"requestedBy"`
	MaxRunSeconds  int64          `json:"maxRunSeconds"`
	// WantDigest asks the executor to keep a bounded sample of the rows for an
	// inline email table. It is only ever set for schedules that render one.
	WantDigest bool `json:"wantDigest,omitempty"`
}

type ExecuteResult struct {
	ArtifactKey       string                 `json:"artifactKey"`
	RowCount          int64                  `json:"rowCount"`
	ByteSize          int64                  `json:"byteSize"`
	Truncated         bool                   `json:"truncated"`
	CacheHit          bool                   `json:"cacheHit"`
	ArtifactExpiresAt int64                  `json:"artifactExpiresAt,omitempty"`
	Digest            *services.ReportDigest `json:"digest,omitempty"`
}

type FinalizePayload struct {
	RunID             pulid.ID         `json:"runId"`
	OrganizationID    pulid.ID         `json:"organizationId"`
	BusinessUnitID    pulid.ID         `json:"businessUnitId"`
	Status            report.RunStatus `json:"status"`
	Error             *report.RunError `json:"error,omitempty"`
	ArtifactKey       string           `json:"artifactKey,omitempty"`
	CacheHit          bool             `json:"cacheHit"`
	ArtifactExpiresAt int64            `json:"artifactExpiresAt,omitempty"`
	RowCount          int64            `json:"rowCount"`
	ByteSize          int64            `json:"byteSize"`
	Truncated         bool             `json:"truncated"`
	DurationMs        int64            `json:"durationMs"`
}

type RunReportResult struct {
	RunID     pulid.ID         `json:"runId"`
	Status    report.RunStatus `json:"status"`
	RowCount  int64            `json:"rowCount"`
	ByteSize  int64            `json:"byteSize"`
	Truncated bool             `json:"truncated"`
}

type DeliverRunPayload struct {
	// Digest rides the workflow rather than the database: it is only ever needed
	// for the delivery that immediately follows this run.
	Digest *services.ReportDigest `json:"digest,omitempty"`

	RunID          pulid.ID `json:"runId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type DeliverRunResult struct {
	EmailedRecipients int    `json:"emailedRecipients"`
	EmailAttached     bool   `json:"emailAttached"`
	EmailError        string `json:"emailError,omitempty"`
	NotifiedUsers     int    `json:"notifiedUsers"`
	Skipped           bool   `json:"skipped"`
	// AlertSkipReason explains a delivery the alert condition held back.
	AlertSkipReason string `json:"alertSkipReason,omitempty"`
}

type CleanupExpiredResult struct {
	DeletedArtifacts int `json:"deletedArtifacts"`
	ExpiredRuns      int `json:"expiredRuns"`
}

type ReconcileZombiesResult struct {
	ZombieRuns int `json:"zombieRuns"`
}
