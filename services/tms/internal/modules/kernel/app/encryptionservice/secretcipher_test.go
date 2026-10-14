package encryptionservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretCipherBindsTheScope(t *testing.T) {
	t.Parallel()

	cipher := encryptionservice.NewSecretCipher(
		encryptionservice.NewWithKeyManager(
			encryptionservice.NewLocalKeyManager("test-secret-test-secret-test-secret"),
		),
	)
	scope := &kernelports.SecretScope{
		Purpose:    kernelports.SecretPurposeFactoringTransport,
		Tenant:     pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		ResourceID: "factrn_1",
	}
	key := "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----\n"

	sealed, err := cipher.Seal(t.Context(), key, scope)
	require.NoError(t, err)
	assert.NotContains(t, sealed, "OPENSSH")

	opened, err := cipher.Open(t.Context(), sealed, scope)
	require.NoError(t, err)
	assert.Equal(t, key, opened)

	other := *scope
	other.ResourceID = "factrn_2"
	_, err = cipher.Open(t.Context(), sealed, &other)
	require.Error(t, err)
}
