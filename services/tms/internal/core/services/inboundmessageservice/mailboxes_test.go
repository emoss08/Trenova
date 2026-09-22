package inboundmessageservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type adminMailboxRepo struct {
	repositories.InboundMailboxRepository

	stored    *inboundmessage.Mailbox
	createErr error
	updates   []*inboundmessage.Mailbox
}

func (r *adminMailboxRepo) Create(
	_ context.Context,
	entity *inboundmessage.Mailbox,
) (*inboundmessage.Mailbox, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	entity.ID = pulid.MustNew("imbx_")
	r.stored = entity

	return entity, nil
}

func (r *adminMailboxRepo) GetByID(
	_ context.Context,
	req repositories.GetMailboxByIDRequest,
) (*inboundmessage.Mailbox, error) {
	if r.stored == nil || r.stored.ID != req.ID || r.stored.OrganizationID != req.TenantInfo.OrgID {
		return nil, errortypes.NewNotFoundError("Mailbox not found")
	}
	copied := *r.stored

	return &copied, nil
}

func (r *adminMailboxRepo) Update(
	_ context.Context,
	entity *inboundmessage.Mailbox,
) (*inboundmessage.Mailbox, error) {
	r.updates = append(r.updates, entity)
	r.stored = entity

	return entity, nil
}

func adminActor(org, bu pulid.ID) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: org,
		BusinessUnitID: bu,
	}
}

func adminService(repo *adminMailboxRepo) *Service {
	return &Service{l: zap.NewNop(), mailboxRepo: repo, encryption: plainSecrets{}}
}

func resendSettings() MailboxSettings {
	return MailboxSettings{
		Name:          "Tenders",
		Address:       "  Tenders@Acme-Logistics.com ",
		Provider:      inboundmessage.ProviderResend,
		ReviewPolicy:  inboundmessage.ReviewBelowConfidence,
		MinConfidence: 0.8,
		Status:        inboundmessage.MailboxActive,
	}
}

const resendSecret = "whsec_c2lnbmluZy1rZXktZm9yLXRoZS10ZXN0cw=="

/*
A mailbox's token identifies the tenant to anyone who holds it, so only its
hash is stored. Creation is the one response that can show the token — the
returned token must be the one whose hash was stored, or the URL a person
copies would never resolve.
*/
func TestCreateMailbox_StoresOnlyTheHashOfTheTokenItReturns(t *testing.T) {
	t.Parallel()

	repo := &adminMailboxRepo{}
	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")

	created, err := adminService(repo).CreateMailbox(t.Context(), CreateMailboxRequest{
		Actor:         adminActor(org, bu),
		Settings:      resendSettings(),
		SigningSecret: resendSecret,
	})
	require.NoError(t, err)

	require.NotEmpty(t, created.Token)
	assert.Equal(t, hashutils.SHA256Hex(created.Token), repo.stored.TokenHash)
	assert.NotContains(t, repo.stored.TokenHash, created.Token)
	assert.Equal(t, org, repo.stored.OrganizationID)
	assert.Equal(t, bu, repo.stored.BusinessUnitID)
	assert.Equal(t, "tenders@acme-logistics.com", repo.stored.Address)
	assert.Equal(t, sealedPrefix+resendSecret, repo.stored.SigningSecret,
		"the secret is stored sealed, never in the clear")
}

func TestCreateMailbox_WithoutASecretIsCreatedButRefusesDeliveriesUntilOneIsSet(t *testing.T) {
	t.Parallel()

	repo := &adminMailboxRepo{}
	_, err := adminService(repo).CreateMailbox(t.Context(), CreateMailboxRequest{
		Actor:    adminActor(pulid.MustNew("org_"), pulid.MustNew("bu_")),
		Settings: resendSettings(),
	})
	require.NoError(t, err)
	assert.Empty(t, repo.stored.SigningSecret)
}

func TestCreateMailbox_RefusesASecretInTheWrongScheme(t *testing.T) {
	t.Parallel()

	postmark := resendSettings()
	postmark.Provider = inboundmessage.ProviderPostmark

	cases := []struct {
		name     string
		settings MailboxSettings
		secret   string
	}{
		{"a Resend secret without its prefix", resendSettings(), "c2lnbmluZw=="},
		{"a Resend secret that is not base64", resendSettings(), "whsec_!!!"},
		{"a Postmark secret with no password", postmark, "postmark"},
		{"a Postmark password anybody could guess", postmark, "postmark:short"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &adminMailboxRepo{}
			_, err := adminService(repo).CreateMailbox(t.Context(), CreateMailboxRequest{
				Actor:         adminActor(pulid.MustNew("org_"), pulid.MustNew("bu_")),
				Settings:      tc.settings,
				SigningSecret: tc.secret,
			})

			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
			assert.Equal(t, "signingSecret", validation.Field)
			assert.Nil(t, repo.stored, "nothing is created with a secret that would fail every delivery")
		})
	}
}

func TestCreateMailbox_ReportsATakenAddressOnTheAddressField(t *testing.T) {
	t.Parallel()

	repo := &adminMailboxRepo{createErr: &pgconn.PgError{Code: "23505"}}
	_, err := adminService(repo).CreateMailbox(t.Context(), CreateMailboxRequest{
		Actor:    adminActor(pulid.MustNew("org_"), pulid.MustNew("bu_")),
		Settings: resendSettings(),
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "address", validation.Field)
}

func seededAdminMailbox(t *testing.T) (*adminMailboxRepo, *services.RequestActor) {
	t.Helper()

	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")
	repo := &adminMailboxRepo{stored: &inboundmessage.Mailbox{
		ID:             pulid.MustNew("imbx_"),
		OrganizationID: org,
		BusinessUnitID: bu,
		Name:           "Tenders",
		Address:        "tenders@acme-logistics.com",
		Provider:       inboundmessage.ProviderResend,
		TokenHash:      hashutils.SHA256Hex("the-old-token"),
		SigningSecret:  sealedPrefix + resendSecret,
		ReviewPolicy:   inboundmessage.ReviewAlways,
		Status:         inboundmessage.MailboxActive,
		Version:        3,
	}}

	return repo, adminActor(org, bu)
}

/*
A secret belongs to one provider's scheme. Moving a mailbox from Resend to
Postmark with the Resend secret still on it would check Postmark's credentials
against a Svix key — every delivery refused, and nothing on the page saying
why. So the secret is cleared, and the page asks for the new provider's.
*/
func TestUpdateMailbox_ClearsTheSecretWhenTheProviderChanges(t *testing.T) {
	t.Parallel()

	repo, actor := seededAdminMailbox(t)
	settings := resendSettings()
	settings.Provider = inboundmessage.ProviderPostmark

	updated, err := adminService(repo).UpdateMailbox(t.Context(), UpdateMailboxRequest{
		Actor: actor, ID: repo.stored.ID, Version: 3, Settings: settings,
	})
	require.NoError(t, err)
	assert.Empty(t, updated.SigningSecret)
	assert.Equal(t, int64(3), updated.Version, "the version the person edited is the one checked")
}

func TestUpdateMailbox_KeepsTheSecretAndTokenWhenOnlySettingsChange(t *testing.T) {
	t.Parallel()

	repo, actor := seededAdminMailbox(t)
	settings := resendSettings()
	settings.Name = "Tenders and PODs"

	updated, err := adminService(repo).UpdateMailbox(t.Context(), UpdateMailboxRequest{
		Actor: actor, ID: repo.stored.ID, Version: 3, Settings: settings,
	})
	require.NoError(t, err)
	assert.Equal(t, "Tenders and PODs", updated.Name)
	assert.Equal(t, sealedPrefix+resendSecret, updated.SigningSecret)
	assert.Equal(t, hashutils.SHA256Hex("the-old-token"), updated.TokenHash)
}

func TestUpdateMailbox_CannotReachAnotherTenantsMailbox(t *testing.T) {
	t.Parallel()

	repo, _ := seededAdminMailbox(t)
	_, err := adminService(repo).UpdateMailbox(t.Context(), UpdateMailboxRequest{
		Actor:    adminActor(pulid.MustNew("org_"), pulid.MustNew("bu_")),
		ID:       repo.stored.ID,
		Version:  3,
		Settings: resendSettings(),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, repo.updates)
}

func TestRotateMailboxToken_RetiresTheOldTokenAndReturnsTheNewOneOnce(t *testing.T) {
	t.Parallel()

	repo, actor := seededAdminMailbox(t)
	rotated, err := adminService(repo).RotateMailboxToken(t.Context(), MailboxActionRequest{
		Actor: actor, ID: repo.stored.ID,
	})
	require.NoError(t, err)

	assert.NotEqual(t, hashutils.SHA256Hex("the-old-token"), repo.stored.TokenHash)
	assert.Equal(t, hashutils.SHA256Hex(rotated.Token), repo.stored.TokenHash)
	assert.Equal(t, sealedPrefix+resendSecret, repo.stored.SigningSecret, "rotation leaves the secret alone")
}

func TestSetMailboxSigningSecret_SealsASecretInTheMailboxesScheme(t *testing.T) {
	t.Parallel()

	repo, actor := seededAdminMailbox(t)
	svc := adminService(repo)

	_, err := svc.SetMailboxSigningSecret(t.Context(), SetMailboxSecretRequest{
		Actor: actor, ID: repo.stored.ID, Secret: "postmark:a-long-random-webhook-password",
	})
	require.Error(t, err, "a Resend mailbox takes a Resend secret")

	updated, err := svc.SetMailboxSigningSecret(t.Context(), SetMailboxSecretRequest{
		Actor: actor, ID: repo.stored.ID, Secret: "  whsec_bmV3LWtleQ==  ",
	})
	require.NoError(t, err)
	assert.Equal(t, sealedPrefix+"whsec_bmV3LWtleQ==", updated.SigningSecret)
	assert.False(t, strings.Contains(updated.TokenHash, "whsec_"))
}
