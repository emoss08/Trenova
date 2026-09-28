package capture

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	bun.BaseModel             `bun:"table:capture_devices,alias:cdev" json:"-"`
	pagination.CursorValueSet `bun:",embed"                           json:"-"`

	ID                   pulid.ID     `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID       pulid.ID     `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID     `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	UserID               pulid.ID     `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Name                 string       `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	MachineName          string       `json:"machineName"    bun:"machine_name,type:VARCHAR(255),notnull"`
	WindowsUser          string       `json:"windowsUser"    bun:"windows_user,type:VARCHAR(255),nullzero"`
	AgentVersion         string       `json:"agentVersion"   bun:"agent_version,type:VARCHAR(50),notnull"`
	Architecture         Architecture `json:"architecture"   bun:"architecture,type:VARCHAR(10),notnull"`
	OSVersion            string       `json:"osVersion"      bun:"os_version,type:VARCHAR(100),nullzero"`
	Status               DeviceStatus `json:"status"         bun:"status,type:VARCHAR(20),notnull"`
	RefreshTokenHash     string       `json:"-"              bun:"refresh_token_hash,type:VARCHAR(128),notnull"`
	PreviousRefreshHash  string       `json:"-"              bun:"previous_refresh_hash,type:VARCHAR(128),nullzero"`
	AccessTokenHash      string       `json:"-"              bun:"access_token_hash,type:VARCHAR(128),notnull"`
	AccessTokenExpiresAt int64        `json:"-"              bun:"access_token_expires_at,type:BIGINT,notnull"`
	Sources              []SourceInfo `json:"sources"        bun:"sources,type:JSONB,notnull,default:'[]'"`
	LastSeenAt           *int64       `json:"lastSeenAt"     bun:"last_seen_at,type:BIGINT,nullzero"`
	LastIP               string       `json:"lastIp"         bun:"last_ip,type:VARCHAR(64),nullzero"`
	RevokedAt            *int64       `json:"revokedAt"      bun:"revoked_at,type:BIGINT,nullzero"`
	RevokedByID          *pulid.ID    `json:"revokedById"    bun:"revoked_by_id,type:VARCHAR(100),nullzero"`
	RevokedReason        string       `json:"revokedReason"  bun:"revoked_reason,type:VARCHAR(255),nullzero"`
	Version              int64        `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt            int64        `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64        `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
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

func (d *CaptureDevice) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("cdev_")
		}
		if d.Status == "" {
			d.Status = DeviceActive
		}
		if d.Sources == nil {
			d.Sources = []SourceInfo{}
		}
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}

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
