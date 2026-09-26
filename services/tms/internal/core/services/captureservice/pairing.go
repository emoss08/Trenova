package captureservice

import (
	"context"
	"crypto/rand"
	"math/big"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/uptrace/bun"
)

const (
	// userCodeAlphabet is consonants only, per RFC 8628 §6.1: no vowels, so a
	// code never spells a word, and no characters that read alike.
	userCodeAlphabet     = "BCDFGHJKLMNPQRSTVWXZ"
	userCodeAttempts     = 5
	maxDeviceNameLength  = 100
	pairingVerifyPath    = "/capture/pair"
	userCodeGroupSize    = 4
	maxMachineNameLength = 255
)

// PairingErrorCode is one of the RFC 8628 §3.5 token endpoint answers.
type PairingErrorCode string

const (
	PairingAuthorizationPending = PairingErrorCode("authorization_pending")
	PairingSlowDown             = PairingErrorCode("slow_down")
	PairingAccessDenied         = PairingErrorCode("access_denied")
	PairingExpiredToken         = PairingErrorCode("expired_token")
	PairingInvalidGrant         = PairingErrorCode("invalid_grant")
)

// PairingError is how a device learns where its grant is. It is answered as
// an RFC 8628 error body so any OAuth device-flow client understands it.
type PairingError struct {
	Code PairingErrorCode
}

func (e *PairingError) Error() string {
	return string(e.Code)
}

func pairingError(code PairingErrorCode) error {
	return &PairingError{Code: code}
}

type StartPairingRequest struct {
	MachineName  string               `json:"machineName"`
	WindowsUser  string               `json:"windowsUser"`
	AgentVersion string               `json:"agentVersion"`
	Architecture capture.Architecture `json:"architecture"`
	OSVersion    string               `json:"osVersion"`
	ClientIP     string               `json:"-"`
}

// PairingGrant is what the device shows and holds while it waits. The device
// code never leaves the machine; the user code is what the person types.
type PairingGrant struct {
	DeviceCode              string `json:"deviceCode"`
	UserCode                string `json:"userCode"`
	VerificationURI         string `json:"verificationUri"`
	VerificationURIComplete string `json:"verificationUriComplete"`
	ExpiresIn               int64  `json:"expiresIn"`
	Interval                int64  `json:"interval"`
}

// StartPairing issues a device authorization grant. It is unauthenticated: a
// machine asking for a code cannot yet say who it belongs to, and the grant
// it gets carries no tenant until a signed-in person approves it.
func (s *Service) StartPairing(
	ctx context.Context,
	req *StartPairingRequest,
) (*PairingGrant, error) {
	if err := validateMachine(req.MachineName, req.AgentVersion, req.Architecture); err != nil {
		return nil, err
	}

	deviceCode, deviceCodeHash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	for range userCodeAttempts {
		userCode, codeErr := newUserCode()
		if codeErr != nil {
			return nil, codeErr
		}

		_, err = s.pairings.Create(ctx, &capture.CapturePairing{
			DeviceCodeHash: deviceCodeHash,
			UserCode:       userCode,
			Status:         capture.PairingPending,
			MachineName:    strings.TrimSpace(req.MachineName),
			WindowsUser:    strings.TrimSpace(req.WindowsUser),
			AgentVersion:   strings.TrimSpace(req.AgentVersion),
			Architecture:   req.Architecture,
			OSVersion:      strings.TrimSpace(req.OSVersion),
			ClientIP:       req.ClientIP,
			ExpiresAt:      now + capture.PairingLifetimeSeconds,
		})
		if err == nil {
			display := FormatUserCode(userCode)
			verify := s.cfg.App.GetWebBaseURL() + pairingVerifyPath

			return &PairingGrant{
				DeviceCode:              deviceCode,
				UserCode:                display,
				VerificationURI:         verify,
				VerificationURIComplete: verify + "?code=" + display,
				ExpiresIn:               capture.PairingLifetimeSeconds,
				Interval:                capture.PairingPollIntervalSeconds,
			}, nil
		}
		if !dberror.IsUniqueConstraintViolation(err) {
			return nil, err
		}
	}

	return nil, err
}

func validateMachine(machineName, agentVersion string, arch capture.Architecture) error {
	multiErr := errortypes.NewMultiError()
	name := strings.TrimSpace(machineName)
	if name == "" {
		multiErr.Add("machineName", errortypes.ErrRequired, "Machine name is required")
	} else if len(name) > maxMachineNameLength {
		multiErr.Add("machineName", errortypes.ErrInvalid, "Machine name is too long")
	}
	if strings.TrimSpace(agentVersion) == "" {
		multiErr.Add("agentVersion", errortypes.ErrRequired, "Agent version is required")
	}
	if !arch.IsValid() {
		multiErr.Add("architecture", errortypes.ErrInvalid, "Architecture must be x64")
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func newUserCode() (string, error) {
	var b strings.Builder
	b.Grow(capture.UserCodeLength)

	limit := big.NewInt(int64(len(userCodeAlphabet)))
	for range capture.UserCodeLength {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		b.WriteByte(userCodeAlphabet[n.Int64()])
	}

	return b.String(), nil
}

// FormatUserCode shows a code the way people read it: two groups of four.
func FormatUserCode(code string) string {
	if len(code) != capture.UserCodeLength {
		return code
	}

	return code[:userCodeGroupSize] + "-" + code[userCodeGroupSize:]
}

// NormalizeUserCode accepts a code however it was typed: lower case, with or
// without the dash, with stray spaces.
func NormalizeUserCode(raw string) string {
	var b strings.Builder
	b.Grow(capture.UserCodeLength)
	for _, r := range strings.ToUpper(raw) {
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// PairingPreview is what the approving person is shown: which machine is
// asking, so a code read off somebody else's screen is recognisably not
// theirs.
type PairingPreview struct {
	UserCode     string               `json:"userCode"`
	MachineName  string               `json:"machineName"`
	WindowsUser  string               `json:"windowsUser"`
	AgentVersion string               `json:"agentVersion"`
	Architecture capture.Architecture `json:"architecture"`
	OSVersion    string               `json:"osVersion"`
	ClientIP     string               `json:"clientIp"`
	ExpiresAt    int64                `json:"expiresAt"`
}

// PreviewPairing shows a pending grant to the person about to approve it.
func (s *Service) PreviewPairing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userCode string,
) (*PairingPreview, error) {
	if _, err := s.require(
		ctx,
		tenantInfo,
		permission.ResourceCaptureBatch,
		permission.OpCreate,
	); err != nil {
		return nil, err
	}

	pairing, err := s.openPairing(ctx, userCode)
	if err != nil {
		return nil, err
	}

	return &PairingPreview{
		UserCode:     FormatUserCode(pairing.UserCode),
		MachineName:  pairing.MachineName,
		WindowsUser:  pairing.WindowsUser,
		AgentVersion: pairing.AgentVersion,
		Architecture: pairing.Architecture,
		OSVersion:    pairing.OSVersion,
		ClientIP:     pairing.ClientIP,
		ExpiresAt:    pairing.ExpiresAt,
	}, nil
}

// openPairing finds a grant a person can still act on. Every miss reads the
// same, so a guessed code tells the guesser nothing about which codes exist.
func (s *Service) openPairing(
	ctx context.Context,
	userCode string,
) (*capture.CapturePairing, error) {
	code := NormalizeUserCode(userCode)
	notFound := errortypes.NewNotFoundError("That code is not valid or has expired")
	if len(code) != capture.UserCodeLength {
		return nil, notFound
	}

	pairing, err := s.pairings.GetOpenByUserCode(ctx, code)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, notFound
		}

		return nil, err
	}
	if pairing.Status != capture.PairingPending || pairing.IsExpired(timeutils.NowUnix()) {
		return nil, notFound
	}

	return pairing, nil
}

type DecidePairingRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	UserCode   string                `json:"userCode"`
	DeviceName string                `json:"deviceName"`
	Approve    bool                  `json:"approve"`
}

// DecidePairing approves or denies a grant. Approval binds it to the person
// approving and the organization they are signed in to; nothing the machine
// sent decides either.
func (s *Service) DecidePairing(ctx context.Context, req *DecidePairingRequest) error {
	if _, err := s.require(
		ctx,
		req.TenantInfo,
		permission.ResourceCaptureBatch,
		permission.OpCreate,
	); err != nil {
		return err
	}
	if req.Approve {
		if _, err := s.requireEnabled(ctx, req.TenantInfo); err != nil {
			return err
		}
	}

	pairing, err := s.openPairing(ctx, req.UserCode)
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	pairing.DecidedAt = &now
	if !req.Approve {
		pairing.Status = capture.PairingDenied
		_, err = s.pairings.Update(ctx, pairing)

		return err
	}

	name := strings.TrimSpace(req.DeviceName)
	if name == "" {
		name = pairing.MachineName
	}
	if len(name) > maxDeviceNameLength {
		return errortypes.NewValidationError("deviceName", errortypes.ErrInvalid,
			"Device name must be at most {0} characters", maxDeviceNameLength)
	}

	orgID, buID, userID := req.TenantInfo.OrgID, req.TenantInfo.BuID, req.TenantInfo.UserID
	pairing.Status = capture.PairingApproved
	pairing.OrganizationID = &orgID
	pairing.BusinessUnitID = &buID
	pairing.ApprovedByID = &userID
	pairing.DeviceName = name

	_, err = s.pairings.Update(ctx, pairing)

	return err
}

// ExchangePairing is the device polling its grant. On an approved grant it
// creates the device and returns its first credential, exactly once.
func (s *Service) ExchangePairing(ctx context.Context, deviceCode string) (*TokenPair, error) {
	if strings.TrimSpace(deviceCode) == "" {
		return nil, pairingError(PairingInvalidGrant)
	}

	pairing, err := s.pairings.GetByDeviceCodeHash(ctx, tokenutils.Hash(deviceCode))
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, pairingError(PairingInvalidGrant)
		}

		return nil, err
	}

	now := timeutils.NowUnix()
	if denial := pollOutcome(pairing, now); denial != "" {
		if pairing.Status == capture.PairingPending && denial != PairingExpiredToken {
			pairing.LastPolledAt = &now
			// A poll that lost a race with another poll is not a failure: the
			// other one already recorded the time.
			if _, err = s.pairings.Update(ctx, pairing); err != nil &&
				!errortypes.IsVersionMismatchError(err) {
				return nil, err
			}
		}

		return nil, pairingError(denial)
	}

	var pair *TokenPair
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		device, issued, issueErr := newDeviceFromPairing(pairing, now)
		if issueErr != nil {
			return issueErr
		}

		created, createErr := s.devices.Create(txCtx, device)
		if createErr != nil {
			return createErr
		}

		pairing.Status = capture.PairingConsumed
		pairing.DeviceID = &created.ID
		if _, updateErr := s.pairings.Update(txCtx, pairing); updateErr != nil {
			return updateErr
		}

		pair = issued.forDevice(created)

		return nil
	})
	if err != nil {
		// Two polls exchanging the same approval race here; the loser finds
		// the grant consumed, which is what a replayed code must see too.
		if dberror.IsConstraintViolation(err) || errortypes.IsVersionMismatchError(err) {
			return nil, pairingError(PairingInvalidGrant)
		}

		return nil, err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  pair.OrganizationID,
		BuID:   pair.BusinessUnitID,
		UserID: pair.UserID,
	}
	s.logAudit(&services.LogActionParams{
		Resource:       permission.ResourceCaptureDevice,
		ResourceID:     pair.DeviceID.String(),
		Operation:      permission.OpCreate,
		UserID:         pair.UserID,
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pair.UserID,
		OrganizationID: pair.OrganizationID,
		BusinessUnitID: pair.BusinessUnitID,
		Critical:       true,
	}, "Paired Trenova Capture on "+pairing.MachineName)
	s.publish(
		ctx,
		tenantInfo,
		permission.ResourceCaptureDevice,
		pair.DeviceID,
		"created",
		pair.UserID,
		nil,
	)

	return pair, nil
}

// pollOutcome is what a poll of this grant answers, empty when the grant is
// ready to exchange.
func pollOutcome(pairing *capture.CapturePairing, now int64) PairingErrorCode {
	switch pairing.Status {
	case capture.PairingDenied:
		return PairingAccessDenied
	case capture.PairingConsumed:
		return PairingInvalidGrant
	case capture.PairingExpired:
		return PairingExpiredToken
	case capture.PairingPending, capture.PairingApproved:
	}

	if pairing.IsExpired(now) {
		return PairingExpiredToken
	}
	if pairing.Status == capture.PairingPending {
		if pairing.PolledTooSoon(now) {
			return PairingSlowDown
		}

		return PairingAuthorizationPending
	}
	if pairing.OrganizationID == nil || pairing.BusinessUnitID == nil ||
		pairing.ApprovedByID == nil {
		return PairingInvalidGrant
	}

	return ""
}

func newDeviceFromPairing(
	pairing *capture.CapturePairing,
	now int64,
) (*capture.CaptureDevice, *issuedTokens, error) {
	issued, err := issueTokens(now)
	if err != nil {
		return nil, nil, err
	}

	device := &capture.CaptureDevice{
		ID:                   pulid.MustNew("cdev_"),
		OrganizationID:       *pairing.OrganizationID,
		BusinessUnitID:       *pairing.BusinessUnitID,
		UserID:               *pairing.ApprovedByID,
		Name:                 pairing.DeviceName,
		MachineName:          pairing.MachineName,
		WindowsUser:          pairing.WindowsUser,
		AgentVersion:         pairing.AgentVersion,
		Architecture:         pairing.Architecture,
		OSVersion:            pairing.OSVersion,
		Status:               capture.DeviceActive,
		RefreshTokenHash:     issued.refreshHash,
		AccessTokenHash:      issued.accessHash,
		AccessTokenExpiresAt: issued.accessExpiresAt,
		LastSeenAt:           &now,
		LastIP:               pairing.ClientIP,
		Sources:              []capture.SourceInfo{},
	}
	if device.Name == "" {
		device.Name = pairing.MachineName
	}

	multiErr := errortypes.NewMultiError()
	device.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, nil, multiErr
	}

	return device, issued, nil
}

// ExpirePairings closes grants nobody approved in time, freeing their codes.
func (s *Service) ExpirePairings(ctx context.Context) (int, error) {
	return s.pairings.ExpireStale(ctx, timeutils.NowUnix())
}
