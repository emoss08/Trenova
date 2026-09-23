package inboundmessageservice

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/emoss08/trenova/shared/webhooksig"
	"go.uber.org/zap"
)

// minPostmarkPasswordLength is the shortest basic-auth password a Postmark
// mailbox accepts. The password is the whole of Postmark's proof that a
// delivery is genuine, so it has to be one nobody can guess.
const minPostmarkPasswordLength = 16

// MailboxSettings is everything about a mailbox a person configures. The token
// and the signing secret are not settings: the token is minted, never chosen,
// and the secret is set on its own so it is never echoed back in a form.
type MailboxSettings struct {
	Name          string
	Address       string
	Provider      inboundmessage.Provider
	Purpose       string
	ReviewPolicy  inboundmessage.ReviewPolicy
	MinConfidence float64
	Status        inboundmessage.MailboxStatus
}

type CreateMailboxRequest struct {
	Actor    *services.RequestActor
	Settings MailboxSettings
	// SigningSecret is optional at creation: a mailbox without one refuses
	// every delivery until it is set, which is the safe way to be half done.
	SigningSecret string
}

type UpdateMailboxRequest struct {
	Actor    *services.RequestActor
	ID       pulid.ID
	Version  int64
	Settings MailboxSettings
}

type MailboxActionRequest struct {
	Actor *services.RequestActor
	ID    pulid.ID
}

type SetMailboxSecretRequest struct {
	Actor  *services.RequestActor
	ID     pulid.ID
	Secret string
}

// MailboxCredentials is a mailbox with its webhook token, returned only when
// the token is minted. It is the one moment the token exists in the clear.
type MailboxCredentials struct {
	Mailbox *inboundmessage.Mailbox
	Token   string
}

func (m MailboxSettings) apply(mailbox *inboundmessage.Mailbox) {
	mailbox.Name = strings.TrimSpace(m.Name)
	mailbox.Address = strings.ToLower(strings.TrimSpace(m.Address))
	mailbox.Provider = m.Provider
	mailbox.Purpose = strings.TrimSpace(m.Purpose)
	mailbox.ReviewPolicy = m.ReviewPolicy
	mailbox.MinConfidence = m.MinConfidence
	mailbox.Status = m.Status
}

// validateSigningSecret checks a secret is in the provider's scheme before it
// is sealed. A secret in the wrong shape would be stored happily and then fail
// every delivery, which reads to the person who set it as mail that never came.
func validateSigningSecret(provider inboundmessage.Provider, secret string) error {
	switch provider {
	case inboundmessage.ProviderResend:
		key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
		if !strings.HasPrefix(secret, "whsec_") || err != nil || len(key) == 0 {
			return errortypes.NewValidationError("signingSecret", errortypes.ErrInvalid,
				"A Resend signing secret starts with whsec_, as Resend shows it")
		}
	case inboundmessage.ProviderPostmark:
		_, password, ok := webhooksig.ParseBasicCredentials(secret)
		if !ok {
			return errortypes.NewValidationError("signingSecret", errortypes.ErrInvalid,
				"A Postmark secret is the user:password pair in the webhook URL")
		}
		if len(password) < minPostmarkPasswordLength {
			return errortypes.NewValidationError("signingSecret", errortypes.ErrInvalid,
				"The password must be at least {0} characters", minPostmarkPasswordLength)
		}
	default:
		return errortypes.NewValidationError("provider", errortypes.ErrInvalid,
			"Provider must be Postmark or Resend")
	}

	return nil
}

func (s *Service) sealSecret(provider inboundmessage.Provider, secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if err := validateSigningSecret(provider, secret); err != nil {
		return "", err
	}

	sealed, err := s.encryption.EncryptString(secret)
	if err != nil {
		return "", fmt.Errorf("seal the signing secret: %w", err)
	}

	return sealed, nil
}

func validateMailbox(mailbox *inboundmessage.Mailbox) error {
	multiErr := errortypes.NewMultiError()
	mailbox.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// addressTaken turns the address index's refusal into the field error it is.
func addressTaken(err error) error {
	if dberror.IsUniqueConstraintViolation(err) {
		return errortypes.NewValidationError("address", errortypes.ErrDuplicate,
			"Another mailbox already listens on this address")
	}

	return err
}

func (s *Service) getMailbox(
	ctx context.Context,
	id pulid.ID,
	tenant pagination.TenantInfo,
) (*inboundmessage.Mailbox, error) {
	return s.mailboxRepo.GetByID(
		ctx,
		repositories.GetMailboxByIDRequest{ID: id, TenantInfo: tenant},
	)
}

// CreateMailbox mints the mailbox's webhook token and returns it once. Only its
// hash is stored, so this is the one response that can show it.
func (s *Service) CreateMailbox(
	ctx context.Context,
	req CreateMailboxRequest,
) (*MailboxCredentials, error) {
	tenant := req.Actor.TenantInfo()
	mailbox := &inboundmessage.Mailbox{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
	req.Settings.apply(mailbox)
	mailbox.ApplyDefaults()

	if err := validateMailbox(mailbox); err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.SigningSecret) != "" {
		sealed, err := s.sealSecret(mailbox.Provider, req.SigningSecret)
		if err != nil {
			return nil, err
		}
		mailbox.SigningSecret = sealed
	}

	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}
	mailbox.TokenHash = tokenHash

	created, err := s.mailboxRepo.Create(ctx, mailbox)
	if err != nil {
		return nil, addressTaken(err)
	}

	s.auditMailbox(req.Actor, permission.OpCreate, nil, created, "Mailbox created")

	return &MailboxCredentials{Mailbox: created, Token: token}, nil
}

// UpdateMailbox changes a mailbox's settings. Changing its provider clears the
// signing secret, because the old one belongs to the other provider's scheme;
// the mailbox refuses deliveries until the new provider's secret is set,
// rather than verifying the new provider's mail against the old one's secret.
func (s *Service) UpdateMailbox(
	ctx context.Context,
	req UpdateMailboxRequest,
) (*inboundmessage.Mailbox, error) {
	tenant := req.Actor.TenantInfo()
	original, err := s.getMailbox(ctx, req.ID, tenant)
	if err != nil {
		return nil, err
	}

	mailbox := *original
	req.Settings.apply(&mailbox)
	mailbox.Version = req.Version
	if mailbox.Provider != original.Provider {
		mailbox.SigningSecret = ""
	}

	if err = validateMailbox(&mailbox); err != nil {
		return nil, err
	}

	updated, err := s.mailboxRepo.Update(ctx, &mailbox)
	if err != nil {
		return nil, addressTaken(err)
	}

	s.auditMailbox(req.Actor, permission.OpUpdate, original, updated, "Mailbox updated")

	return updated, nil
}

// RotateMailboxToken replaces the webhook token. The old URL stops working the
// moment this returns, so it is what to do when a token has leaked — and the
// provider has to be pointed at the new URL before mail flows again.
func (s *Service) RotateMailboxToken(
	ctx context.Context,
	req MailboxActionRequest,
) (*MailboxCredentials, error) {
	original, err := s.getMailbox(ctx, req.ID, req.Actor.TenantInfo())
	if err != nil {
		return nil, err
	}

	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}

	mailbox := *original
	mailbox.TokenHash = tokenHash

	updated, err := s.mailboxRepo.Update(ctx, &mailbox)
	if err != nil {
		return nil, err
	}

	s.auditMailbox(
		req.Actor,
		permission.OpUpdate,
		original,
		updated,
		"Mailbox webhook token rotated",
	)

	return &MailboxCredentials{Mailbox: updated, Token: token}, nil
}

// SetMailboxSigningSecret seals and stores the provider's secret. It is never
// returned: a person who needs it again copies it from the provider.
func (s *Service) SetMailboxSigningSecret(
	ctx context.Context,
	req SetMailboxSecretRequest,
) (*inboundmessage.Mailbox, error) {
	original, err := s.getMailbox(ctx, req.ID, req.Actor.TenantInfo())
	if err != nil {
		return nil, err
	}

	sealed, err := s.sealSecret(original.Provider, req.Secret)
	if err != nil {
		return nil, err
	}

	mailbox := *original
	mailbox.SigningSecret = sealed

	updated, err := s.mailboxRepo.Update(ctx, &mailbox)
	if err != nil {
		return nil, err
	}

	s.auditMailbox(req.Actor, permission.OpUpdate, original, updated, "Mailbox signing secret set")

	return updated, nil
}

// auditMailbox records a configuration change. The token hash and the sealed
// secret are not serialised with the mailbox, so neither reaches the log.
func (s *Service) auditMailbox(
	actor *services.RequestActor,
	op permission.Operation,
	before, after *inboundmessage.Mailbox,
	comment string,
) {
	if s.audit == nil {
		return
	}

	auditActor := actor.AuditActor()
	params := &services.LogActionParams{
		Resource:       permission.ResourceInboundMailbox,
		ResourceID:     after.ID.String(),
		Operation:      op,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(after),
		OrganizationID: after.OrganizationID,
		BusinessUnitID: after.BusinessUnitID,
	}
	options := []services.LogOption{auditservice.WithComment(comment)}
	if before != nil {
		params.PreviousState = jsonutils.MustToJSON(before)
		options = append(options, auditservice.WithDiff(before, after))
	}

	if err := s.audit.LogAction(params, options...); err != nil {
		s.l.Error("failed to log mailbox audit action",
			zap.String("mailboxId", after.ID.String()), zap.Error(err))
	}
}
