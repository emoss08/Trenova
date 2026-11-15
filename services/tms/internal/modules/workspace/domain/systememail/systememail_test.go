package systememail_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAPIKey = "pm-0123456789abcdef0123456789"

func credential() *systememail.Credential {
	return &systememail.Credential{
		Provider:    email.ProviderPostmark,
		APIKey:      testAPIKey,
		FromAddress: "no-reply@mail.trenova.app",
		FromName:    "Trenova",
	}
}

func TestTheFingerprintMatchesTheContractVectors(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"97bc40decf9416a4926287451b37e8a9c5e5d55b2b142126e28a0993d604d01f",
		systememail.Fingerprint(credential()),
	)
	assert.Equal(
		t,
		"1ca13b2b81836149a7d6d52423e8725b17510d2e80953be5e82ef90a0c9736fc",
		systememail.Fingerprint(&systememail.Credential{
			Provider:      email.ProviderResend,
			APIKey:        "re_0123456789abcdef0123456789",
			FromAddress:   "no-reply@mail.trenova.app",
			FromName:      "Trenova",
			WebhookSecret: "whsec_abcdef",
		}),
	)
}

func TestTheFingerprintChangesWhenAnyFieldChanges(t *testing.T) {
	t.Parallel()

	base := systememail.Fingerprint(credential())
	assert.Equal(t, base, systememail.Fingerprint(credential()))

	changes := map[string]func(c *systememail.Credential){
		"provider":      func(c *systememail.Credential) { c.Provider = email.ProviderResend },
		"apiKey":        func(c *systememail.Credential) { c.APIKey += "x" },
		"fromAddress":   func(c *systememail.Credential) { c.FromAddress = "mail@mail.trenova.app" },
		"fromName":      func(c *systememail.Credential) { c.FromName = "Trenova Cloud" },
		"webhookSecret": func(c *systememail.Credential) { c.WebhookSecret = "whsec_1" },
	}
	for field, change := range changes {
		c := credential()
		change(c)
		assert.NotEqual(t, base, systememail.Fingerprint(c), field)
	}
}

func TestACredentialNeverPrintsItsSecrets(t *testing.T) {
	t.Parallel()

	c := credential()
	c.WebhookSecret = "whsec_do-not-print"
	for _, printed := range []string{
		fmt.Sprintf("%v", c),
		fmt.Sprintf("%+v", c),
		fmt.Sprintf("%#v", c),
		fmt.Sprintf("%s", c),
	} {
		assert.NotContains(t, printed, testAPIKey)
		assert.NotContains(t, printed, "whsec_do-not-print")
	}
}

func TestAValidCredentialPasses(t *testing.T) {
	t.Parallel()

	require.NoError(t, credential().Validate())
}

func TestAMalformedCredentialIsRefusedWithoutEchoingTheSecret(t *testing.T) {
	t.Parallel()

	cases := map[string]func(c *systememail.Credential){
		"unknown provider":    func(c *systememail.Credential) { c.Provider = "SES" },
		"short key":           func(c *systememail.Credential) { c.APIKey = "short" },
		"key with a space":    func(c *systememail.Credential) { c.APIKey = testAPIKey + " x" },
		"key with a newline":  func(c *systememail.Credential) { c.APIKey = testAPIKey + "\nx" },
		"oversized key":       func(c *systememail.Credential) { c.APIKey = strings.Repeat("k", 513) },
		"missing key":         func(c *systememail.Credential) { c.APIKey = "" },
		"bad from address":    func(c *systememail.Credential) { c.FromAddress = "not an address" },
		"named from address":  func(c *systememail.Credential) { c.FromAddress = "Trenova <a@b.co>" },
		"empty from name":     func(c *systememail.Credential) { c.FromName = "" },
		"long from name":      func(c *systememail.Credential) { c.FromName = strings.Repeat("n", 41) },
		"padded from name":    func(c *systememail.Credential) { c.FromName = " Trenova" },
		"control in name":     func(c *systememail.Credential) { c.FromName = "Tre\nnova" },
		"bad webhook secret":  func(c *systememail.Credential) { c.WebhookSecret = "a b" },
		"long webhook secret": func(c *systememail.Credential) { c.WebhookSecret = strings.Repeat("w", 513) },
	}
	for name, change := range cases {
		c := credential()
		change(c)
		err := c.Validate()
		require.Error(t, err, name)
		assert.NotContains(t, err.Error(), testAPIKey, name)
	}
}

func TestTheDisplayNameSaysTheMailIsSentByTrenovaForTheOrganization(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Acme Freight via Trenova", systememail.DisplayName("Acme Freight", "Trenova"))
	assert.Equal(t, "Trenova", systememail.DisplayName("  ", "Trenova"))

	long := systememail.DisplayName(strings.Repeat("a", 200), "Trenova")
	assert.LessOrEqual(t, len([]rune(long)), 100)
	assert.True(t, strings.HasSuffix(long, " via Trenova"))
}
