package capture

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	// PairingLifetimeSeconds is how long a person has to approve a code. Long
	// enough to sign in first, short enough that a code read over a shoulder is
	// useless by the time it could be typed somewhere else.
	PairingLifetimeSeconds = 10 * 60
	// PairingPollIntervalSeconds is how often the companion may ask whether its
	// code was approved. Asking faster earns a slow_down, per RFC 8628.
	PairingPollIntervalSeconds = 5
	// UserCodeLength is the code a person types, without its separator.
	UserCodeLength = 8
)

// CapturePairing is one device authorization grant (RFC 8628): a code shown on the
// machine, approved by a signed-in person in the browser, and exchanged by the
// machine for its credential.
//
// It has no tenant until it is approved. The machine asking for a code does not
// know, and must not be able to choose, which organization it will belong to:
// that is decided by who approves it and where they are signed in.
type CapturePairing struct {
	bun.BaseModel `bun:"table:capture_pairings,alias:cpair" json:"-"`

	ID             pulid.ID      `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	DeviceCodeHash string        `json:"-"              bun:"device_code_hash,type:VARCHAR(128),notnull"`
	UserCode       string        `json:"userCode"       bun:"user_code,type:VARCHAR(16),notnull"`
	Status         PairingStatus `json:"status"         bun:"status,type:VARCHAR(20),notnull"`
	MachineName    string        `json:"machineName"    bun:"machine_name,type:VARCHAR(255),notnull"`
	WindowsUser    string        `json:"windowsUser"    bun:"windows_user,type:VARCHAR(255),nullzero"`
	AgentVersion   string        `json:"agentVersion"   bun:"agent_version,type:VARCHAR(50),notnull"`
	Architecture   Architecture  `json:"architecture"   bun:"architecture,type:VARCHAR(10),notnull"`
	OSVersion      string        `json:"osVersion"      bun:"os_version,type:VARCHAR(100),nullzero"`
	ClientIP       string        `json:"clientIp"       bun:"client_ip,type:VARCHAR(64),nullzero"`
	OrganizationID *pulid.ID     `json:"organizationId" bun:"organization_id,type:VARCHAR(100),nullzero"`
	BusinessUnitID *pulid.ID     `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),nullzero"`
	ApprovedByID   *pulid.ID     `json:"approvedById"   bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	DeviceName     string        `json:"deviceName"     bun:"device_name,type:VARCHAR(100),nullzero"`
	DeviceID       *pulid.ID     `json:"deviceId"       bun:"device_id,type:VARCHAR(100),nullzero"`
	LastPolledAt   *int64        `json:"-"              bun:"last_polled_at,type:BIGINT,nullzero"`
	ExpiresAt      int64         `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	DecidedAt      *int64        `json:"decidedAt"      bun:"decided_at,type:BIGINT,nullzero"`
	Version        int64         `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64         `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64         `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// IsExpired reports whether the code can no longer be approved or exchanged.
func (p *CapturePairing) IsExpired(now int64) bool {
	return now >= p.ExpiresAt
}

// PolledTooSoon reports whether the companion asked again before the interval
// it was given. The poll is recorded either way, so a companion that ignores
// slow_down keeps being told to.
func (p *CapturePairing) PolledTooSoon(now int64) bool {
	return p.LastPolledAt != nil && now-*p.LastPolledAt < PairingPollIntervalSeconds
}

func (p *CapturePairing) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("cpair_")
		}
		if p.Status == "" {
			p.Status = PairingPending
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

func (p *CapturePairing) GetID() pulid.ID      { return p.ID }
func (p *CapturePairing) GetCreatedAt() int64  { return p.CreatedAt }
func (p *CapturePairing) GetTableName() string { return "capture_pairings" }
