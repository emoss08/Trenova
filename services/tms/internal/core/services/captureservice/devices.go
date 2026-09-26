package captureservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/emoss08/trenova/shared/versionutils"
	"go.uber.org/zap"
)

const (
	// AccessTokenLifetimeSeconds keeps a stolen access token short-lived; the
	// refresh that renews it is where revocation and the user's standing are
	// checked.
	AccessTokenLifetimeSeconds = 15 * 60
	// refreshIdleLimitSeconds ends a device nobody has used in three months.
	// A laptop in a drawer should not come back to life holding a credential.
	refreshIdleLimitSeconds = 90 * 24 * 60 * 60
	// touchIntervalSeconds bounds how often an authenticated request writes
	// the device's last-seen time.
	touchIntervalSeconds  = 30
	accessTokenPrefix     = "tcd_at_"
	refreshTokenPrefix    = "tcd_rt_"
	maxSources            = 50
	maxAgentVersionLength = 50
	maxOSVersionLength    = 100
	maxSourceNameLength   = 255
	maxRevokeReason       = 255
	reasonReplayed        = "The device's credential was presented twice; it may have been copied"
	reasonIdle            = "The device was not used for 90 days"
)

// OutdatedAgentError refuses a companion older than the organization allows.
// It is its own type so the handler can answer 426 and the companion can
// offer to update instead of showing a generic failure.
type OutdatedAgentError struct {
	Current string
	Minimum string
	// AutoUpdate is whether the organization lets the companion update
	// itself, so it knows to start the update rather than say to ask IT.
	AutoUpdate bool
}

func (e *OutdatedAgentError) Error() string {
	return "Trenova Capture " + e.Current + " is older than the minimum " + e.Minimum +
		" this organization allows; update it to continue"
}

// TokenPair is a device's credential. The refresh token is shown once, here,
// and stored only as its hash.
type TokenPair struct {
	TokenType            string   `json:"tokenType"`
	AccessToken          string   `json:"accessToken"`
	AccessTokenExpiresAt int64    `json:"accessTokenExpiresAt"`
	RefreshToken         string   `json:"refreshToken"`
	DeviceID             pulid.ID `json:"deviceId"`
	DeviceName           string   `json:"deviceName"`
	UserID               pulid.ID `json:"userId"`
	OrganizationID       pulid.ID `json:"organizationId"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"`
}

type issuedTokens struct {
	access          string
	accessHash      string
	accessExpiresAt int64
	refresh         string
	refreshHash     string
}

func issueTokens(now int64) (*issuedTokens, error) {
	access, _, err := tokenutils.New()
	if err != nil {
		return nil, err
	}
	refresh, _, err := tokenutils.New()
	if err != nil {
		return nil, err
	}

	access = accessTokenPrefix + access
	refresh = refreshTokenPrefix + refresh

	return &issuedTokens{
		access:          access,
		accessHash:      tokenutils.Hash(access),
		accessExpiresAt: now + AccessTokenLifetimeSeconds,
		refresh:         refresh,
		refreshHash:     tokenutils.Hash(refresh),
	}, nil
}

func (t *issuedTokens) forDevice(device *capture.CaptureDevice) *TokenPair {
	return &TokenPair{
		TokenType:            "Bearer",
		AccessToken:          t.access,
		AccessTokenExpiresAt: t.accessExpiresAt,
		RefreshToken:         t.refresh,
		DeviceID:             device.ID,
		DeviceName:           device.Name,
		UserID:               device.UserID,
		OrganizationID:       device.OrganizationID,
		BusinessUnitID:       device.BusinessUnitID,
	}
}

// DevicePrincipal is an authenticated device and the person it acts for.
type DevicePrincipal struct {
	Device *capture.CaptureDevice
}

// TenantInfo is the device's person, in their organization.
func (p *DevicePrincipal) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.Device.OrganizationID,
		BuID:   p.Device.BusinessUnitID,
		UserID: p.Device.UserID,
	}
}

var errInvalidDeviceToken = errortypes.NewAuthenticationError("The device credential is not valid")

// Authenticate resolves an access token to its device. Every failure reads
// the same, so the endpoint says nothing about why a token was refused.
func (s *Service) Authenticate(
	ctx context.Context,
	accessToken string,
	clientIP string,
) (*DevicePrincipal, error) {
	if !strings.HasPrefix(accessToken, accessTokenPrefix) {
		return nil, errInvalidDeviceToken
	}

	device, err := s.devices.GetByAccessTokenHash(ctx, tokenutils.Hash(accessToken))
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errInvalidDeviceToken
		}

		return nil, err
	}

	now := timeutils.NowUnix()
	if !device.IsActive() || now >= device.AccessTokenExpiresAt {
		return nil, errInvalidDeviceToken
	}

	principal := &DevicePrincipal{Device: device}
	s.Heartbeat(ctx, principal, clientIP)

	return principal, nil
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
	AgentVersion string `json:"agentVersion"`
	OSVersion    string `json:"osVersion"`
	ClientIP     string `json:"-"`
}

// Refresh rotates a device's credential. It is where a device is held to its
// person's current standing: a revoked device, a person who left, capture
// turned off, or a companion too old all stop here.
func (s *Service) Refresh(ctx context.Context, req *RefreshRequest) (*TokenPair, error) {
	if !strings.HasPrefix(req.RefreshToken, refreshTokenPrefix) {
		return nil, pairingError(PairingInvalidGrant)
	}

	hash := tokenutils.Hash(req.RefreshToken)
	device, err := s.devices.GetByRefreshTokenHash(ctx, hash)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			return nil, err
		}

		return nil, s.handleRefreshMiss(ctx, hash)
	}

	now := timeutils.NowUnix()
	if !device.IsActive() {
		return nil, pairingError(PairingInvalidGrant)
	}
	if device.LastSeenAt != nil && now-*device.LastSeenAt > refreshIdleLimitSeconds {
		s.revoke(ctx, device, nil, reasonIdle, now)

		return nil, pairingError(PairingInvalidGrant)
	}

	tenantInfo := (&DevicePrincipal{Device: device}).TenantInfo()
	control, err := s.requireEnabled(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	agentVersion := strings.TrimSpace(req.AgentVersion)
	if agentVersion == "" {
		agentVersion = device.AgentVersion
	}
	if len(agentVersion) > maxAgentVersionLength || len(req.OSVersion) > maxOSVersionLength {
		return nil, errortypes.NewValidationError("agentVersion", errortypes.ErrInvalid,
			"Agent or OS version is too long")
	}
	if !versionutils.AtLeast(agentVersion, control.CaptureMinAgentVersion) {
		return nil, &OutdatedAgentError{
			Current:    agentVersion,
			Minimum:    control.CaptureMinAgentVersion,
			AutoUpdate: control.CaptureAllowAutoUpdate,
		}
	}

	if _, err = s.require(
		ctx,
		tenantInfo,
		permission.ResourceCaptureBatch,
		permission.OpCreate,
	); err != nil {
		return nil, err
	}

	issued, err := issueTokens(now)
	if err != nil {
		return nil, err
	}

	device.PreviousRefreshHash = device.RefreshTokenHash
	device.RefreshTokenHash = issued.refreshHash
	device.AccessTokenHash = issued.accessHash
	device.AccessTokenExpiresAt = issued.accessExpiresAt
	device.AgentVersion = agentVersion
	if os := strings.TrimSpace(req.OSVersion); os != "" {
		device.OSVersion = os
	}
	device.LastSeenAt = &now
	device.LastIP = req.ClientIP

	updated, err := s.devices.Update(ctx, device)
	if err != nil {
		// Two refreshes with the same token race here. The loser's token is
		// now the previous one, so its next attempt is treated as reuse: the
		// companion serializes refreshes, and one that does not is broken.
		if errortypes.IsVersionMismatchError(err) {
			return nil, pairingError(PairingInvalidGrant)
		}

		return nil, err
	}

	return issued.forDevice(updated), nil
}

// handleRefreshMiss looks for the token among replaced ones. Finding it means
// the credential exists in two places, and the device is revoked so neither
// copy works: the person pairs again, which is cheap, and whoever copied it is
// locked out, which is the point.
func (s *Service) handleRefreshMiss(ctx context.Context, hash string) error {
	device, err := s.devices.GetByPreviousRefreshHash(ctx, hash)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return pairingError(PairingInvalidGrant)
		}

		return err
	}

	if device.IsActive() {
		s.l.Warn("capture device refresh token reused; revoking device",
			zap.String("deviceId", device.ID.String()),
			zap.String("organizationId", device.OrganizationID.String()))
		s.revoke(ctx, device, nil, reasonReplayed, timeutils.NowUnix())
	}

	return pairingError(PairingInvalidGrant)
}

// revoke ends a device, records why, and closes its stream.
func (s *Service) revoke(
	ctx context.Context,
	device *capture.CaptureDevice,
	by *pulid.ID,
	reason string,
	now int64,
) *capture.CaptureDevice {
	device.Revoke(by, reason, now)

	updated, err := s.devices.Update(ctx, device)
	if err != nil {
		s.l.Error("failed to revoke capture device",
			zap.String("deviceId", device.ID.String()), zap.Error(err))

		return nil
	}

	actor := updated.UserID
	principal := services.PrincipalTypeSystem
	if by != nil {
		actor = *by
		principal = services.PrincipalTypeUser
	}
	s.logAudit(&services.LogActionParams{
		Resource:       permission.ResourceCaptureDevice,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         actor,
		PrincipalType:  principal,
		PrincipalID:    actor,
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		Critical:       true,
	}, "Revoked capture device: "+reason)

	tenantInfo := pagination.TenantInfo{
		OrgID:  updated.OrganizationID,
		BuID:   updated.BusinessUnitID,
		UserID: actor,
	}
	s.publish(
		ctx,
		tenantInfo,
		permission.ResourceCaptureDevice,
		updated.ID,
		"revoked",
		updated.UserID,
		DeviceSignal{DeviceID: updated.ID},
	)

	return updated
}

// DeviceSignal is the entity carried on a realtime event meant for a device's
// stream. It names the device so the stream can ignore events for the same
// person's other devices.
type DeviceSignal struct {
	DeviceID  pulid.ID `json:"deviceId"`
	RequestID pulid.ID `json:"requestId,omitempty"`
}

type RevokeDeviceRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	DeviceID   pulid.ID              `json:"deviceId"`
	Reason     string                `json:"reason"`
	// Mine limits the revoke to the caller's own device, and a device that is
	// somebody else's reads as not found rather than asking for permission.
	Mine bool `json:"mine"`
}

// RevokeDevice ends a device. A person may always revoke their own; revoking
// somebody else's takes capture device update.
func (s *Service) RevokeDevice(
	ctx context.Context,
	req *RevokeDeviceRequest,
) (*capture.CaptureDevice, error) {
	device, err := s.devices.GetByID(ctx, repositories.GetCaptureDeviceByIDRequest{
		ID:         req.DeviceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if device.UserID != req.TenantInfo.UserID {
		if req.Mine {
			return nil, errortypes.NewNotFoundError("Capture device not found")
		}
		if _, err = s.require(
			ctx,
			req.TenantInfo,
			permission.ResourceCaptureDevice,
			permission.OpUpdate,
		); err != nil {
			return nil, err
		}
	}
	if !device.IsActive() {
		return device, nil
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Revoked from Trenova"
	}
	if len(reason) > maxRevokeReason {
		return nil, errortypes.NewValidationError("reason", errortypes.ErrInvalid,
			"Reason must be at most {0} characters", maxRevokeReason)
	}

	by := req.TenantInfo.UserID
	revoked := s.revoke(ctx, device, &by, reason, timeutils.NowUnix())
	if revoked == nil {
		return nil, errortypes.NewConflictError(
			"The device changed while it was being revoked; try again",
		)
	}

	return revoked, nil
}

type ListDevicesRequest struct {
	TenantInfo pagination.TenantInfo    `json:"-"`
	Filter     *pagination.QueryOptions `json:"-"`
	// Mine lists the caller's own devices, which needs no permission beyond
	// being able to capture. Everyone's takes capture device read.
	Mine   bool                 `json:"mine"`
	Status capture.DeviceStatus `json:"status"`
}

func (s *Service) ListDevices(
	ctx context.Context,
	req *ListDevicesRequest,
) (*pagination.ListResult[*capture.CaptureDevice], error) {
	repoReq := &repositories.ListCaptureDevicesRequest{Filter: req.Filter, Status: req.Status}
	if req.Mine {
		repoReq.UserID = req.TenantInfo.UserID
	} else if _, err := s.require(ctx, req.TenantInfo, permission.ResourceCaptureDevice, permission.OpRead); err != nil {
		return nil, err
	}

	return s.devices.List(ctx, repoReq)
}

// ReportSources records the scanners a device can reach, so the web app only
// offers what is actually attached.
func (s *Service) ReportSources(
	ctx context.Context,
	principal *DevicePrincipal,
	sources []capture.SourceInfo,
) (*capture.CaptureDevice, error) {
	if len(sources) > maxSources {
		return nil, errortypes.NewValidationError("sources", errortypes.ErrInvalid,
			"A device may report at most {0} scanners", maxSources)
	}

	multiErr := errortypes.NewMultiError()
	for i, source := range sources {
		name := strings.TrimSpace(source.Name)
		if name == "" || len(name) > maxSourceNameLength {
			multiErr.WithIndex("sources", i).Add("name", errortypes.ErrInvalid,
				"Scanner name must be 1 to 255 characters")
		}
		if !source.Protocol.IsValid() {
			multiErr.WithIndex("sources", i).Add("protocol", errortypes.ErrInvalid,
				"Protocol must be TWAIN or WIA")
		}
		if source.Bitness != 32 && source.Bitness != 64 {
			multiErr.WithIndex("sources", i).Add("bitness", errortypes.ErrInvalid,
				"Bitness must be 32 or 64")
		}
		sources[i].Name = name
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	device := principal.Device
	device.Sources = sources

	return s.devices.Update(ctx, device)
}

// Heartbeat records that a device with an open stream is still there, at
// most once per touch interval.
func (s *Service) Heartbeat(ctx context.Context, principal *DevicePrincipal, clientIP string) {
	device := principal.Device
	now := timeutils.NowUnix()
	if device.LastSeenAt != nil && now-*device.LastSeenAt < touchIntervalSeconds {
		return
	}

	if err := s.devices.Touch(ctx, repositories.TouchCaptureDeviceRequest{
		ID: device.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: device.OrganizationID,
			BuID:  device.BusinessUnitID,
		},
		SeenAt: now,
		IP:     clientIP,
	}); err != nil {
		s.l.Warn(
			"could not record a device heartbeat",
			zap.String("deviceId", device.ID.String()),
			zap.Error(err),
		)

		return
	}
	device.LastSeenAt = &now
}
