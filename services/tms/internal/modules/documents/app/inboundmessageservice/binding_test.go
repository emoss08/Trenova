package inboundmessageservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func boundSecretService(repo *adminMailboxRepo) (*Service, *encryptionservice.Service) {
	enc := encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("inbound-mailbox-binding-test-key-32+chars"),
	)
	return &Service{l: zap.NewNop(), mailboxRepo: repo, encryption: enc}, enc
}

func TestCreateMailbox_BindsTheSigningSecretToItsMailbox(t *testing.T) {
	t.Parallel()

	repo := &adminMailboxRepo{}
	svc, enc := boundSecretService(repo)
	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")

	_, err := svc.CreateMailbox(t.Context(), CreateMailboxRequest{
		Actor:         adminActor(org, bu),
		Settings:      resendSettings(),
		SigningSecret: resendSecret,
	})
	require.NoError(t, err)
	stored := repo.stored
	require.False(t, stored.ID.IsNil())

	opened, err := svc.openMailboxSecret(
		stored,
		inboundmessage.SigningSecretField,
		stored.SigningSecret,
	)
	require.NoError(t, err)
	assert.Equal(t, resendSecret, opened)

	_, err = enc.DecryptString(stored.SigningSecret)
	require.Error(t, err, "the secret no longer opens without its mailbox binding")

	other := *stored
	other.ID = pulid.MustNew("imbx_")
	_, err = svc.openMailboxSecret(&other, inboundmessage.SigningSecretField, stored.SigningSecret)
	require.Error(t, err, "a secret copied onto another mailbox does not open there")

	_, err = svc.openMailboxSecret(stored, inboundmessage.ProviderAPIKeyField, stored.SigningSecret)
	require.Error(t, err, "a signing secret does not open as the API key")
}

func TestOpenMailboxSecret_StillOpensASecretSealedBeforeBinding(t *testing.T) {
	t.Parallel()

	svc, enc := boundSecretService(&adminMailboxRepo{})
	legacy, err := enc.EncryptString(resendSecret)
	require.NoError(t, err)
	mailbox := &inboundmessage.Mailbox{
		ID:             pulid.MustNew("imbx_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	opened, err := svc.openMailboxSecret(mailbox, inboundmessage.SigningSecretField, legacy)

	require.NoError(t, err)
	assert.Equal(t, resendSecret, opened)
}
