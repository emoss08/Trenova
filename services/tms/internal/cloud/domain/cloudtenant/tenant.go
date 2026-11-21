package cloudtenant

import (
	"time"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type State string

const (
	StateApplying  = State("applying")
	StateActive    = State("active")
	StateSuspended = State("suspended")
	StateRemoving  = State("removing")
	StateRestoring = State("restoring")
	StateRemoved   = State("removed")
)

const RefusalInvalid = "invalid"

type Tenant struct {
	keel.Table `keel:"cloud_tenants"`

	InstanceID           string   `json:"instanceId"           keel:"instance_id,pk"`
	OrganizationID       pulid.ID `json:"organizationId"       keel:"tenant_organization_id,nullzero"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"       keel:"tenant_business_unit_id,nullzero"`
	AdminUserID          pulid.ID `json:"adminUserId"          keel:"admin_user_id,nullzero"`
	State                State    `json:"state"                keel:"state"`
	AppliedRevision      int64    `json:"appliedRevision"      keel:"applied_revision"`
	DesiredRevision      int64    `json:"desiredRevision"      keel:"desired_revision"`
	DesiredDigest        string   `json:"desiredDigest"        keel:"desired_digest"`
	Desired              Desired  `json:"desired"              keel:"desired,jsonb"`
	InviteSHA256         string   `json:"inviteSha256"         keel:"admin_invite_sha256,nullzero"`
	LastArchiveCommandID string   `json:"lastArchiveCommandId" keel:"last_archive_command_id,nullzero"`
	LastArchivePurpose   string   `json:"lastArchivePurpose"   keel:"last_archive_purpose,nullzero"`
	LastArchiveLocation  string   `json:"lastArchiveLocation"  keel:"last_archive_location,nullzero"`
	LastArchiveSizeBytes int64    `json:"lastArchiveSizeBytes" keel:"last_archive_size_bytes,nullzero"`
	LastArchiveSHA256    string   `json:"lastArchiveSha256"    keel:"last_archive_sha256,nullzero"`
	LastArchiveTakenAt   int64    `json:"lastArchiveTakenAt"   keel:"last_archive_taken_at,nullzero"`
	LastRestoreCommandID string   `json:"lastRestoreCommandId" keel:"last_restore_command_id,nullzero"`
	LastRestoreState     string   `json:"lastRestoreState"     keel:"last_restore_state,nullzero"`
	LastRestoreAt        int64    `json:"lastRestoreAt"        keel:"last_restore_at,nullzero"`
	LastRestoreReason    string   `json:"lastRestoreReason"    keel:"last_restore_reason,nullzero"`
	ImportedAt           int64    `json:"importedAt"           keel:"imported_at,nullzero"`
	RefusedRevision      int64    `json:"refusedRevision"      keel:"refused_revision,nullzero"`
	RefusedCode          string   `json:"refusedCode"          keel:"refused_code,nullzero"`
	RefusedReason        string   `json:"refusedReason"        keel:"refused_reason,nullzero"`
	RemovalStartedAt     int64    `json:"removalStartedAt"     keel:"removal_started_at,nullzero"`
	RemovedAt            int64    `json:"removedAt"            keel:"removed_at,nullzero"`
	CreatedAt            int64    `json:"createdAt"            keel:"created_at"`
	UpdatedAt            int64    `json:"updatedAt"            keel:"updated_at"`
}

type Event struct {
	keel.Table `keel:"cloud_tenant_events"`

	ID         pulid.ID       `json:"id"         keel:"id,pk,autoid:ctev_"`
	InstanceID string         `json:"instanceId" keel:"instance_id"`
	Kind       EventKind      `json:"kind"       keel:"kind"`
	Revision   int64          `json:"revision"   keel:"revision"`
	Detail     map[string]any `json:"detail"     keel:"detail,jsonb"`
	OccurredAt int64          `json:"occurredAt" keel:"occurred_at"`
}

type EventKind string

const (
	EventCreated   = EventKind("created")
	EventApplied   = EventKind("applied")
	EventRefused   = EventKind("refused")
	EventSuspended = EventKind("suspended")
	EventResumed   = EventKind("resumed")
	EventRemoving  = EventKind("removing")
	EventRemoved   = EventKind("removed")
	EventArchived  = EventKind("archived")
	EventRestoring = EventKind("restoring")
	EventRestored  = EventKind("restored")
	EventImported  = EventKind("imported")
)

type Refusal struct {
	Revision int64  `json:"revision"`
	Code     string `json:"code"`
	Reason   string `json:"reason"`
}

func Invalid(revision int64, reason string) *Refusal {
	if utf8.RuneCountInString(reason) > maxReasonLength {
		runes := []rune(reason)
		reason = string(runes[:maxReasonLength-1]) + "…"
	}
	return &Refusal{Revision: revision, Code: RefusalInvalid, Reason: reason}
}

func (t *Tenant) Refuse(r *Refusal) {
	if r == nil {
		t.RefusedRevision, t.RefusedCode, t.RefusedReason = 0, "", ""
		return
	}
	t.RefusedRevision, t.RefusedCode, t.RefusedReason = r.Revision, r.Code, r.Reason
}

func (t *Tenant) Refusal() *Refusal {
	if t.RefusedRevision == 0 {
		return nil
	}
	return &Refusal{Revision: t.RefusedRevision, Code: t.RefusedCode, Reason: t.RefusedReason}
}

func (t *Tenant) Made() bool {
	return !t.OrganizationID.IsNil()
}

func (t *Tenant) HasFinalArchive() bool {
	return t.LastArchiveCommandID != "" && t.LastArchivePurpose == ArchivePurposeFinal
}

type Archive struct {
	CommandID string    `json:"commandId"`
	Location  string    `json:"location"`
	SizeBytes int64     `json:"sizeBytes"`
	SHA256    string    `json:"sha256"`
	TakenAt   time.Time `json:"takenAt"`
}

type Restore struct {
	CommandID string    `json:"commandId"`
	State     string    `json:"state"`
	At        time.Time `json:"at"`
	Reason    string    `json:"reason,omitempty"`
}

const (
	RestoreRunning = "running"
	RestoreDone    = "done"
	RestoreFailed  = "failed"
)

type Administrator struct {
	InviteTokenSHA256 string `json:"inviteTokenSha256,omitempty"`
	SetUp             bool   `json:"setUp"`
}

type CommandFailure struct {
	CommandID string `json:"commandId"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason"`
}

type SupportGrant struct {
	ID               string    `json:"id"`
	OrganizationID   string    `json:"organizationId"`
	OrganizationName string    `json:"organizationName"`
	StaffEmail       string    `json:"staffEmail"`
	Scope            string    `json:"scope"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type Status struct {
	InstanceID      string           `json:"instanceId"`
	AppliedRevision *int64           `json:"appliedRevision,omitempty"`
	State           State            `json:"state"`
	Administrator   *Administrator   `json:"administrator,omitempty"`
	LastArchive     *Archive         `json:"lastArchive,omitempty"`
	LastRestore     *Restore         `json:"lastRestore,omitempty"`
	CommandFailures []CommandFailure `json:"commandFailures,omitempty"`
	SupportGrants   *[]SupportGrant  `json:"supportGrants,omitempty"`
	Refused         *Refusal         `json:"refused,omitempty"`
}

func (t *Tenant) Status(admin *Administrator) *Status {
	out := &Status{
		InstanceID:    t.InstanceID,
		State:         t.State,
		Administrator: admin,
		Refused:       t.Refusal(),
	}
	switch t.State {
	case StateRemoving:
		out.State = StateApplying
	case StateRestoring:
		out.State = StateSuspended
	case StateApplying, StateActive, StateSuspended, StateRemoved:
	}
	if t.AppliedRevision > 0 {
		applied := t.AppliedRevision
		out.AppliedRevision = &applied
	}
	if t.LastArchiveCommandID != "" {
		out.LastArchive = &Archive{
			CommandID: t.LastArchiveCommandID,
			Location:  t.LastArchiveLocation,
			SizeBytes: t.LastArchiveSizeBytes,
			SHA256:    t.LastArchiveSHA256,
			TakenAt:   time.Unix(t.LastArchiveTakenAt, 0).UTC(),
		}
	}
	if t.LastRestoreCommandID != "" && t.LastRestoreState != RestoreRunning {
		out.LastRestore = &Restore{
			CommandID: t.LastRestoreCommandID,
			State:     t.LastRestoreState,
			At:        time.Unix(t.LastRestoreAt, 0).UTC(),
			Reason:    t.LastRestoreReason,
		}
	}
	return out
}

func (t *Tenant) Restoring() bool {
	return t.State == StateRestoring
}
