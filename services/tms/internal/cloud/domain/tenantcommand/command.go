package tenantcommand

import (
	"time"

	"github.com/emoss08/keel"
)

type Kind string

const (
	KindArchive = Kind("archive")
	KindRestore = Kind("restore")
	KindImport  = Kind("import")
)

type State string

const (
	StateQueued  = State("queued")
	StateRunning = State("running")
	StateDone    = State("done")
	StateFailed  = State("failed")
)

func (s State) Settled() bool {
	return s == StateDone || s == StateFailed
}

type Phase string

const (
	PhaseQueued    = Phase("queued")
	PhaseTables    = Phase("tables")
	PhaseDocuments = Phase("documents")
	PhaseManifest  = Phase("manifest")
	PhaseChecking  = Phase("checking")
	PhaseSuspend   = Phase("suspending")
	PhasePurging   = Phase("purging")
	PhaseLoading   = Phase("loading")
	PhaseVerifying = Phase("verifying")
	PhaseResuming  = Phase("resuming")
	PhaseFinished  = Phase("finished")
)

type Progress struct {
	Phase          Phase `json:"phase"`
	TablesDone     int   `json:"tablesDone"`
	TablesTotal    int   `json:"tablesTotal"`
	DocumentsDone  int   `json:"documentsDone"`
	DocumentsTotal int   `json:"documentsTotal"`
	Bytes          int64 `json:"bytes"`
}

type Command struct {
	keel.Table `keel:"cloud_tenant_commands"`

	CommandID    string   `json:"commandId"    keel:"command_id,pk"`
	InstanceID   string   `json:"instanceId"   keel:"instance_id"`
	Kind         Kind     `json:"kind"         keel:"kind"`
	Purpose      string   `json:"purpose"      keel:"purpose,nullzero"`
	State        State    `json:"state"        keel:"state"`
	Location     string   `json:"location"     keel:"location"`
	Prefix       string   `json:"prefix"       keel:"prefix"`
	ExpectSHA256 string   `json:"expectSha256" keel:"expect_sha256,nullzero"`
	SealedAccess string   `json:"-"            keel:"sealed_access,nullzero"`
	Progress     Progress `json:"progress"     keel:"progress,jsonb"`
	SHA256       string   `json:"sha256"       keel:"result_sha256,nullzero"`
	SizeBytes    int64    `json:"sizeBytes"    keel:"result_size_bytes,nullzero"`
	TakenAt      int64    `json:"takenAt"      keel:"taken_at,nullzero"`
	FailureCode  string   `json:"failureCode"  keel:"failure_code,nullzero"`
	Reason       string   `json:"reason"       keel:"reason,nullzero"`
	RequestedAt  int64    `json:"requestedAt"  keel:"requested_at"`
	StartedAt    int64    `json:"startedAt"    keel:"started_at,nullzero"`
	FinishedAt   int64    `json:"finishedAt"   keel:"finished_at,nullzero"`
	UpdatedAt    int64    `json:"updatedAt"    keel:"updated_at"`
}

type View struct {
	CommandID    string     `json:"commandId"`
	InstanceID   string     `json:"instanceId"`
	Kind         Kind       `json:"kind"`
	Purpose      string     `json:"purpose,omitempty"`
	State        State      `json:"state"`
	Location     string     `json:"location"`
	Progress     Progress   `json:"progress"`
	SHA256       string     `json:"sha256,omitempty"`
	SizeBytes    int64      `json:"sizeBytes,omitempty"`
	TakenAt      *time.Time `json:"takenAt,omitempty"`
	FailureCode  string     `json:"failureCode,omitempty"`
	Reason       string     `json:"reason,omitempty"`
	RequestedAt  time.Time  `json:"requestedAt"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	ExpectSHA256 string     `json:"expectSha256,omitempty"`
}

func (c *Command) View() *View {
	out := &View{
		CommandID:    c.CommandID,
		InstanceID:   c.InstanceID,
		Kind:         c.Kind,
		Purpose:      c.Purpose,
		State:        c.State,
		Location:     c.Location,
		Progress:     c.Progress,
		SHA256:       c.SHA256,
		SizeBytes:    c.SizeBytes,
		FailureCode:  c.FailureCode,
		Reason:       c.Reason,
		RequestedAt:  time.Unix(c.RequestedAt, 0).UTC(),
		ExpectSHA256: c.ExpectSHA256,
	}
	if c.TakenAt > 0 {
		taken := time.Unix(c.TakenAt, 0).UTC()
		out.TakenAt = &taken
	}
	if c.FinishedAt > 0 {
		finished := time.Unix(c.FinishedAt, 0).UTC()
		out.FinishedAt = &finished
	}
	return out
}

type Grant struct {
	keel.Table `keel:"cloud_archive_grants"`

	Name         string `json:"name"      keel:"name,pk"`
	Location     string `json:"location"  keel:"location"`
	SealedAccess string `json:"-"         keel:"sealed_access"`
	ExpiresAt    int64  `json:"expiresAt" keel:"expires_at"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at"`
}

const NightlyGrant = "nightly"
