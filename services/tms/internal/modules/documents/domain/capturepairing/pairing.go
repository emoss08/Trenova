package capturepairing

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/shared/pulid"
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
	keel.Table `keel:"capture_pairings"`

	ID             pulid.ID             `json:"id"             keel:"id,pk,autoid:cpair_"`
	DeviceCodeHash string               `json:"-"              keel:"device_code_hash"`
	UserCode       string               `json:"userCode"       keel:"user_code"`
	Status         PairingStatus        `json:"status"         keel:"status"`
	MachineName    string               `json:"machineName"    keel:"machine_name"`
	WindowsUser    string               `json:"windowsUser"    keel:"windows_user,nullzero"`
	AgentVersion   string               `json:"agentVersion"   keel:"agent_version"`
	Architecture   capture.Architecture `json:"architecture"   keel:"architecture"`
	OSVersion      string               `json:"osVersion"      keel:"os_version,nullzero"`
	ClientIP       string               `json:"clientIp"       keel:"client_ip,nullzero"`
	OrganizationID *pulid.ID            `json:"organizationId" keel:"organization_id"`
	BusinessUnitID *pulid.ID            `json:"businessUnitId" keel:"business_unit_id"`
	ApprovedByID   *pulid.ID            `json:"approvedById"   keel:"approved_by_id"`
	DeviceName     string               `json:"deviceName"     keel:"device_name,nullzero"`
	DeviceID       *pulid.ID            `json:"deviceId"       keel:"device_id"`
	LastPolledAt   *int64               `json:"-"              keel:"last_polled_at"`
	ExpiresAt      int64                `json:"expiresAt"      keel:"expires_at"`
	DecidedAt      *int64               `json:"decidedAt"      keel:"decided_at"`
	Version        int64                `json:"version"        keel:"version,version"`
	CreatedAt      int64                `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64                `json:"updatedAt"      keel:"updated_at,autoupdate"`
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

func (p *CapturePairing) GetID() pulid.ID      { return p.ID }
func (p *CapturePairing) GetCreatedAt() int64  { return p.CreatedAt }
func (p *CapturePairing) GetTableName() string { return "capture_pairings" }

func (p *CapturePairing) BeforeCreate(ctx context.Context) error {
	if p.Status == "" {
		p.Status = PairingPending
	}

	return nil
}
