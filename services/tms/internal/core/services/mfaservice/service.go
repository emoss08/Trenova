package mfaservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/qrutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/emoss08/trenova/shared/totputils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultIssuer            = "Trenova"
	defaultAuthenticatorName = "Authenticator app"
	qrCodeSize               = 240
	mfaAssuranceLevel        = 2
	auditFieldType           = "type"
)

var (
	errInvalidCode = errortypes.NewValidationError(
		"code",
		errortypes.ErrInvalid,
		"The code is incorrect or has already been used",
	)
	errInvalidRecoveryCode = errortypes.NewValidationError(
		"recoveryCode",
		errortypes.ErrInvalid,
		"The recovery code is incorrect or has already been used",
	)
	errInvalidPassword = errortypes.NewValidationError(
		"password",
		errortypes.ErrInvalid,
		"The password is incorrect",
	)
	errNoActiveFactor = errortypes.NewBusinessError(
		"Two-factor authentication is not turned on for this account",
	)
)

type Params struct {
	fx.In

	Repository repositories.MFARepository
	Users      repositories.UserRepository
	Sessions   repositories.SessionRepository
	Encryption *encryptionservice.Service
	Auditor    services.SecurityAuditor
	Config     *config.Config
	Logger     *zap.Logger
}

type Service struct {
	repo     repositories.MFARepository
	users    repositories.UserRepository
	sessions repositories.SessionRepository
	enc      *encryptionservice.Service
	auditor  services.SecurityAuditor
	issuer   string
	l        *zap.Logger
	now      func() time.Time
}

func New(p Params) services.MFAService { //nolint:gocritic // fx params are passed by value
	return newService(&p)
}

func newService(p *Params) *Service {
	issuer := defaultIssuer
	if p.Config != nil && strings.TrimSpace(p.Config.App.Name) != "" {
		issuer = strings.TrimSpace(p.Config.App.Name)
	}

	return &Service{
		repo:     p.Repository,
		users:    p.Users,
		sessions: p.Sessions,
		enc:      p.Encryption,
		auditor:  p.Auditor,
		issuer:   issuer,
		l:        p.Logger.Named("service.mfa"),
		now:      time.Now,
	}
}

func (s *Service) Status(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sessionAAL int,
) (*services.MFAStatus, error) {
	status := &services.MFAStatus{SessionVerified: sessionAAL >= mfaAssuranceLevel}

	authenticator, err := s.repo.GetTOTP(ctx, tenantInfo.UserID)
	if err != nil {
		return nil, err
	}
	if authenticator == nil {
		return status, nil
	}

	if !authenticator.IsActiveTOTP() {
		status.EnrollmentPending = true
		return status, nil
	}

	status.TOTPEnabled = true
	status.EnabledAt = authenticator.VerifiedAt
	status.LastUsedAt = authenticator.LastUsedAt

	remaining, err := s.repo.CountUnusedRecoveryCodes(ctx, tenantInfo.UserID)
	if err != nil {
		return nil, err
	}
	status.RecoveryCodesRemaining = remaining

	return status, nil
}

func (s *Service) BeginTOTPEnrollment(
	ctx context.Context,
	req *services.BeginTOTPEnrollmentRequest,
) (*services.TOTPEnrollment, error) {
	usr, err := s.verifyPassword(ctx, req.TenantInfo.UserID, req.Password)
	if err != nil {
		return nil, err
	}

	secret, err := totputils.GenerateSecret()
	if err != nil {
		return nil, err
	}

	cipher, err := s.enc.EncryptStringWithAAD(secret, secretAAD(req.TenantInfo.OrgID, usr.ID))
	if err != nil {
		return nil, err
	}

	authenticator := &iam.MFAAuthenticator{
		UserID:         usr.ID,
		OrganizationID: req.TenantInfo.OrgID,
		Type:           iam.MFAAuthenticatorTypeTOTP,
		Name:           defaultAuthenticatorName,
		SecretCipher:   cipher,
	}

	multiErr := errortypes.NewMultiError()
	authenticator.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if err = s.repo.ReplacePendingTOTP(ctx, authenticator); err != nil {
		return nil, err
	}

	uri := totputils.URI(totputils.URIParams{
		Issuer:  s.issuer,
		Account: usr.EmailAddress,
		Secret:  secret,
	})
	qrCode, err := qrutils.PNGDataURI(uri, qrCodeSize)
	if err != nil {
		return nil, err
	}

	return &services.TOTPEnrollment{
		AuthenticatorID: authenticator.ID,
		Secret:          secret,
		URI:             uri,
		QRCode:          qrCode,
	}, nil
}

func (s *Service) ConfirmTOTPEnrollment(
	ctx context.Context,
	req *services.ConfirmTOTPEnrollmentRequest,
) (*services.RecoveryCodesResponse, error) {
	authenticator, err := s.repo.GetTOTP(ctx, req.TenantInfo.UserID)
	if err != nil {
		return nil, err
	}
	if authenticator == nil || authenticator.IsActiveTOTP() {
		return nil, errortypes.NewBusinessError(
			"There is no enrollment waiting for a code. Start the enrollment again.",
		)
	}

	secret, err := s.decryptSecret(authenticator)
	if err != nil {
		return nil, err
	}

	now := s.now()
	step, err := totputils.Verify(totputils.VerifyRequest{
		Secret: secret,
		Code:   req.Code,
		At:     now,
		Skew:   totputils.DefaultSkew,
	})
	if err != nil {
		return nil, errInvalidCode
	}

	codes, hashed, err := s.newRecoveryCodes(req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if err = s.repo.ActivateTOTP(ctx, &repositories.ActivateTOTPRequest{
		AuthenticatorID: authenticator.ID,
		UserID:          req.TenantInfo.UserID,
		Step:            step,
		VerifiedAt:      now.Unix(),
		RecoveryCodes:   hashed,
	}); err != nil {
		return nil, err
	}

	s.raiseSessionAssurance(ctx, req.SessionID, req.TenantInfo.UserID, now.Unix())

	s.recordChange(ctx, &mfaChange{
		tenantInfo: req.TenantInfo,
		targetID:   req.TenantInfo.UserID,
		resourceID: authenticator.ID.String(),
		operation:  permission.OpCreate,
		after:      map[string]any{auditFieldType: iam.MFAAuthenticatorTypeTOTP, "enabled": true},
		comment:    "Turned on two-factor authentication with an authenticator app",
	})

	return &services.RecoveryCodesResponse{RecoveryCodes: codes}, nil
}

func (s *Service) DisableTOTP(ctx context.Context, req *services.DisableTOTPRequest) error {
	if _, err := s.verifyPassword(ctx, req.TenantInfo.UserID, req.Password); err != nil {
		return err
	}

	if _, err := s.VerifySecondFactor(ctx, &services.VerifySecondFactorRequest{
		UserID:       req.TenantInfo.UserID,
		Code:         req.Code,
		RecoveryCode: req.RecoveryCode,
	}); err != nil {
		return err
	}

	removed, err := s.repo.DeleteAll(ctx, req.TenantInfo.UserID)
	if err != nil {
		return err
	}
	if removed == 0 {
		return errNoActiveFactor
	}

	s.recordChange(ctx, &mfaChange{
		tenantInfo: req.TenantInfo,
		targetID:   req.TenantInfo.UserID,
		resourceID: req.TenantInfo.UserID.String(),
		operation:  permission.OpDelete,
		before:     map[string]any{auditFieldType: iam.MFAAuthenticatorTypeTOTP, "enabled": true},
		comment:    "Turned off two-factor authentication",
	})

	return nil
}

func (s *Service) RegenerateRecoveryCodes(
	ctx context.Context,
	req *services.RegenerateRecoveryCodesRequest,
) (*services.RecoveryCodesResponse, error) {
	if err := s.verifyTOTP(ctx, req.TenantInfo.UserID, req.Code); err != nil {
		return nil, err
	}

	codes, hashed, err := s.newRecoveryCodes(req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if err = s.repo.ReplaceRecoveryCodes(ctx, &repositories.ReplaceRecoveryCodesRequest{
		UserID: req.TenantInfo.UserID,
		Codes:  hashed,
	}); err != nil {
		return nil, err
	}

	s.recordChange(ctx, &mfaChange{
		tenantInfo: req.TenantInfo,
		targetID:   req.TenantInfo.UserID,
		resourceID: req.TenantInfo.UserID.String(),
		operation:  permission.OpUpdate,
		comment:    "Replaced two-factor recovery codes",
	})

	return &services.RecoveryCodesResponse{RecoveryCodes: codes}, nil
}

func (s *Service) ResetUserMFA(ctx context.Context, req *services.ResetUserMFARequest) error {
	if req.TargetUserID.IsNil() {
		return errortypes.NewValidationError("userId", errortypes.ErrRequired, "User is required")
	}

	if _, err := s.users.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   req.TenantInfo,
		LookupUserID: req.TargetUserID,
	}); err != nil {
		return err
	}

	removed, err := s.repo.DeleteAll(ctx, req.TargetUserID)
	if err != nil {
		return err
	}
	if removed == 0 {
		return errNoActiveFactor
	}

	s.recordChange(ctx, &mfaChange{
		tenantInfo: req.TenantInfo,
		targetID:   req.TargetUserID,
		resourceID: req.TargetUserID.String(),
		operation:  permission.OpDelete,
		before:     map[string]any{auditFieldType: iam.MFAAuthenticatorTypeTOTP},
		comment:    "An administrator reset the user's two-factor authentication",
	})

	return nil
}

func (s *Service) HasActiveFactor(ctx context.Context, userID pulid.ID) (bool, error) {
	authenticator, err := s.repo.GetTOTP(ctx, userID)
	if err != nil {
		return false, err
	}

	return authenticator.IsActiveTOTP(), nil
}

func (s *Service) VerifySecondFactor(
	ctx context.Context,
	req *services.VerifySecondFactorRequest,
) (string, error) {
	if strings.TrimSpace(req.RecoveryCode) != "" {
		return services.MFAMethodRecoveryCode, s.consumeRecoveryCode(
			ctx,
			req.UserID,
			req.RecoveryCode,
		)
	}

	if err := s.verifyTOTP(ctx, req.UserID, req.Code); err != nil {
		return "", err
	}

	return services.MFAMethodTOTP, nil
}

func (s *Service) Reauthenticate(
	ctx context.Context,
	req *services.ReauthenticateRequest,
) (string, error) {
	if _, err := s.verifyPassword(ctx, req.UserID, req.Password); err != nil {
		return "", err
	}

	active, err := s.HasActiveFactor(ctx, req.UserID)
	if err != nil {
		return "", err
	}
	if !active {
		return "", errNoActiveFactor
	}

	return s.VerifySecondFactor(ctx, &services.VerifySecondFactorRequest{
		UserID:       req.UserID,
		Code:         req.Code,
		RecoveryCode: req.RecoveryCode,
	})
}

func (s *Service) verifyPassword(
	ctx context.Context,
	userID pulid.ID,
	password string,
) (*tenant.User, error) {
	if strings.TrimSpace(password) == "" {
		return nil, errortypes.NewValidationError(
			"password",
			errortypes.ErrRequired,
			"Password is required",
		)
	}

	usr, err := s.users.FindByIDForLogin(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err = usr.VerifyCredentials(password); err != nil {
		if errortypes.IsAuthorizationError(err) {
			return nil, err
		}
		return nil, errInvalidPassword
	}

	return usr, nil
}

func (s *Service) verifyTOTP(ctx context.Context, userID pulid.ID, code string) error {
	if strings.TrimSpace(code) == "" {
		return errortypes.NewValidationError("code", errortypes.ErrRequired, "Code is required")
	}

	authenticator, err := s.repo.GetTOTP(ctx, userID)
	if err != nil {
		return err
	}
	if !authenticator.IsActiveTOTP() {
		return errNoActiveFactor
	}

	secret, err := s.decryptSecret(authenticator)
	if err != nil {
		return err
	}

	now := s.now()
	step, err := totputils.Verify(totputils.VerifyRequest{
		Secret: secret,
		Code:   code,
		At:     now,
		Skew:   totputils.DefaultSkew,
		After:  authenticator.LastUsedStep,
	})
	if err != nil {
		return errInvalidCode
	}

	recorded, err := s.repo.RecordTOTPUse(ctx, repositories.RecordTOTPUseRequest{
		AuthenticatorID: authenticator.ID,
		UserID:          userID,
		Step:            step,
		UsedAt:          now.Unix(),
	})
	if err != nil {
		return err
	}
	if !recorded {
		return errInvalidCode
	}

	return nil
}

func (s *Service) consumeRecoveryCode(ctx context.Context, userID pulid.ID, code string) error {
	normalized := totputils.NormalizeRecoveryCode(code)
	if normalized == "" {
		return errInvalidRecoveryCode
	}

	consumed, err := s.repo.ConsumeRecoveryCode(ctx, repositories.ConsumeRecoveryCodeRequest{
		UserID:   userID,
		CodeHash: tokenutils.Hash(normalized),
		UsedAt:   s.now().Unix(),
	})
	if err != nil {
		return err
	}
	if !consumed {
		return errInvalidRecoveryCode
	}

	return nil
}

func (s *Service) newRecoveryCodes(
	tenantInfo pagination.TenantInfo,
) ([]string, []*iam.MFARecoveryCode, error) {
	codes, err := totputils.GenerateRecoveryCodes(totputils.RecoveryCodeCount)
	if err != nil {
		return nil, nil, err
	}

	hashed := make([]*iam.MFARecoveryCode, 0, len(codes))
	for _, code := range codes {
		hashed = append(hashed, &iam.MFARecoveryCode{
			UserID:         tenantInfo.UserID,
			OrganizationID: tenantInfo.OrgID,
			CodeHash:       tokenutils.Hash(totputils.NormalizeRecoveryCode(code)),
		})
	}

	return codes, hashed, nil
}

func (s *Service) decryptSecret(authenticator *iam.MFAAuthenticator) (string, error) {
	secret, err := s.enc.DecryptStringWithAAD(
		authenticator.SecretCipher,
		secretAAD(authenticator.OrganizationID, authenticator.UserID),
	)
	if err != nil {
		s.l.Error("failed to decrypt a totp secret",
			zap.String("authenticatorID", authenticator.ID.String()),
			zap.Error(err),
		)
		return "", errors.New("two-factor secret could not be read")
	}

	return secret, nil
}

func (s *Service) raiseSessionAssurance(
	ctx context.Context,
	sessionID, userID pulid.ID,
	verifiedAt int64,
) {
	if sessionID.IsNil() || s.sessions == nil {
		return
	}

	sess, err := s.sessions.Get(ctx, sessionID)
	if err != nil || sess == nil || sess.UserID != userID {
		return
	}

	sess.AuthenticatorAAL = max(sess.AuthenticatorAAL, mfaAssuranceLevel)
	sess.MFAAuthenticatedAt = verifiedAt
	sess.LastReauthenticatedAt = verifiedAt
	if err = s.sessions.Update(ctx, sess); err != nil {
		s.l.Warn("failed to raise the session's assurance after enrollment", zap.Error(err))
	}
}

type mfaChange struct {
	tenantInfo pagination.TenantInfo
	targetID   pulid.ID
	resourceID string
	operation  permission.Operation
	before     any
	after      any
	comment    string
}

func (s *Service) recordChange(ctx context.Context, change *mfaChange) {
	if s.auditor == nil {
		return
	}

	s.auditor.RecordChange(ctx, &services.SecurityChange{
		Resource:       permission.ResourceMFAAuthenticator,
		ResourceID:     change.resourceID,
		Operation:      change.operation,
		Actor:          services.UserActor(change.tenantInfo).AuditActor(),
		OrganizationID: change.tenantInfo.OrgID,
		BusinessUnitID: change.tenantInfo.BuID,
		Before:         change.before,
		After:          change.after,
		Comment:        change.comment,
		Metadata:       map[string]any{"userId": change.targetID.String()},
	})
}

func secretAAD(orgID, userID pulid.ID) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeMFATOTPSecret,
		OrganizationID: orgID,
		ResourceID:     userID.String(),
	}
}
