package capture

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	maxDeviceNameLength   = 100
	maxMachineNameLength  = 255
	maxAgentVersionLength = 50
	// OnlineWindowSeconds is how recently a device must have been heard from to
	// be shown as online. The stream heartbeats well inside it, so a device
	// drops off only when it has actually gone away.
	OnlineWindowSeconds = 90
)

// CaptureDevice is one paired Trenova Capture install, acting for one person.
//
// Both tokens are stored hashed. The refresh token rotates on every use, and
// the one it replaced is kept: presenting a replaced token means two parties
// hold the device's credential, and the only safe answer is to revoke it.
type CaptureDevice struct {
	keel.Table `keel:"capture_devices"`

	ID                   pulid.ID     `json:"id"             keel:"id,pk,autoid:cdev_"`
	BusinessUnitID       pulid.ID     `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID     `json:"organizationId" keel:"organization_id,pk,tenant"`
	UserID               pulid.ID     `json:"userId"         keel:"user_id"`
	Name                 string       `json:"name"           keel:"name"`
	MachineName          string       `json:"machineName"    keel:"machine_name"`
	WindowsUser          string       `json:"windowsUser"    keel:"windows_user,nullzero"`
	AgentVersion         string       `json:"agentVersion"   keel:"agent_version"`
	Architecture         Architecture `json:"architecture"   keel:"architecture"`
	OSVersion            string       `json:"osVersion"      keel:"os_version,nullzero"`
	Status               DeviceStatus `json:"status"         keel:"status"`
	RefreshTokenHash     string       `json:"-"              keel:"refresh_token_hash"`
	PreviousRefreshHash  string       `json:"-"              keel:"previous_refresh_hash,nullzero"`
	AccessTokenHash      string       `json:"-"              keel:"access_token_hash"`
	AccessTokenExpiresAt int64        `json:"-"              keel:"access_token_expires_at"`
	Sources              []SourceInfo `json:"sources"        keel:"sources,jsonb"`
	LastSeenAt           *int64       `json:"lastSeenAt"     keel:"last_seen_at"`
	LastIP               string       `json:"lastIp"         keel:"last_ip,nullzero"`
	RevokedAt            *int64       `json:"revokedAt"      keel:"revoked_at"`
	RevokedByID          *pulid.ID    `json:"revokedById"    keel:"revoked_by_id"`
	RevokedReason        string       `json:"revokedReason"  keel:"revoked_reason,nullzero"`
	Version              int64        `json:"version"        keel:"version,version"`
	CreatedAt            int64        `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt            int64        `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
}

// SourceInfo is one scanner the companion can reach, as it last reported it.
type SourceInfo struct {
	Name     string         `json:"name"`
	Protocol SourceProtocol `json:"protocol"`
	// Bitness is 32 or 64: the TWAIN data source's own, which decides which
	// scan helper the companion starts for it.
	Bitness      int  `json:"bitness"`
	IsDefault    bool `json:"isDefault"`
	Duplex       bool `json:"duplex"`
	Feeder       bool `json:"feeder"`
	PatchCodes   bool `json:"patchCodes"`
	Barcodes     bool `json:"barcodes"`
	BlankDiscard bool `json:"blankDiscard"`
	// Resolutions are the DPI values the source accepts, so the web app offers
	// only what the scanner can do.
	Resolutions []int       `json:"resolutions"`
	PixelTypes  []PixelType `json:"pixelTypes"`
}

func (d *CaptureDevice) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.UserID, validation.Required.Error("User is required")),
		validation.Field(&d.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxDeviceNameLength),
		),
		validation.Field(&d.MachineName,
			validation.Required.Error("Machine name is required"),
			validation.Length(1, maxMachineNameLength),
		),
		validation.Field(&d.AgentVersion,
			validation.Required.Error("Agent version is required"),
			validation.Length(1, maxAgentVersionLength),
		),
		validation.Field(&d.Architecture,
			validation.Required.Error("Architecture is required"),
			domainvalidation.ValidEnum[Architecture]("Architecture must be x64"),
		),
		validation.Field(&d.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DeviceStatus]("Status must be Active or Revoked"),
		),
	))
}

// IsActive reports whether the device may still act.
func (d *CaptureDevice) IsActive() bool {
	return d.Status == DeviceActive
}

// IsOnline reports whether the device was heard from recently enough that a
// request sent to it now would be picked up now.
func (d *CaptureDevice) IsOnline(now int64) bool {
	return d.IsActive() && d.LastSeenAt != nil && now-*d.LastSeenAt <= OnlineWindowSeconds
}

// Revoke ends the device. Clearing both hashes means a stolen token stops
// working even if the status check were ever skipped.
func (d *CaptureDevice) Revoke(by *pulid.ID, reason string, now int64) {
	d.Status = DeviceRevoked
	d.RevokedAt = &now
	d.RevokedByID = by
	d.RevokedReason = reason
	d.AccessTokenHash = revokedTokenMarker + d.ID.String()
	d.RefreshTokenHash = revokedTokenMarker + d.ID.String()
	d.PreviousRefreshHash = ""
}

// revokedTokenMarker fills a revoked device's token columns. It is not a hash
// of anything, so no presented token can match it, and it stays unique per
// device so the unique indexes on the columns still hold.
const revokedTokenMarker = "revoked:"

func (d *CaptureDevice) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "cdev",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "machine_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "windows_user",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (d *CaptureDevice) GetID() pulid.ID      { return d.ID }
func (d *CaptureDevice) GetCreatedAt() int64  { return d.CreatedAt }
func (d *CaptureDevice) GetTableName() string { return "capture_devices" }

func (d *CaptureDevice) BeforeCreate(ctx context.Context) error {
	if d.Status == "" {
		d.Status = DeviceActive
	}
	if d.Sources == nil {
		d.Sources = []SourceInfo{}
	}

	return nil
}
