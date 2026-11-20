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

type Administrator struct {
	InviteTokenSHA256 string `json:"inviteTokenSha256,omitempty"`
	SetUp             bool   `json:"setUp"`
}

type Status struct {
	InstanceID      string         `json:"instanceId"`
	AppliedRevision *int64         `json:"appliedRevision,omitempty"`
	State           State          `json:"state"`
	Administrator   *Administrator `json:"administrator,omitempty"`
	LastArchive     *Archive       `json:"lastArchive,omitempty"`
	Refused         *Refusal       `json:"refused,omitempty"`
}

func (t *Tenant) Status(admin *Administrator) *Status {
	out := &Status{
		InstanceID:    t.InstanceID,
		State:         t.State,
		Administrator: admin,
		Refused:       t.Refusal(),
	}
	if t.State == StateRemoving {
		out.State = StateApplying
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
	return out
}
