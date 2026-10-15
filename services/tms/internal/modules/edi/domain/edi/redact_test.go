package edi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactSecrets_KeepsSecretsOutOfQuotes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "the keystore password: [redacted] is sent separately",
		RedactSecrets("the keystore password: Acme#Test2026 is sent separately"))
	assert.Equal(t, "Shared key = [redacted]", RedactSecrets("Shared key = s3cr3t"))
	assert.Equal(t, "cert: [redacted key or certificate]",
		RedactSecrets("cert: -----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"))
	assert.Equal(t,
		"ISA*01*XXXXXXXXXX*06*XXXXXXXXXX*ZZ*ACME",
		RedactSecrets("ISA*01*LOGIN     *06*PASSWORD12*ZZ*ACME"),
	)
	assert.Equal(t, "ISA*00*          *00*          *ZZ*ACME",
		RedactSecrets("ISA*00*          *00*          *ZZ*ACME"), "empty authorization is kept")
}

func TestRedactSecrets_ChangesNothingItAlreadyRedacted(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"the keystore password: Acme#Test2026 is sent separately",
		"token is abc123 and the secret = s3cr3t",
		"cert: -----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
		"ISA*01*LOGIN     *06*PASSWORD12*ZZ*ACME",
		"Trading Partner Code: ACMERETAIL",
	} {
		once := RedactSecrets(text)
		assert.Equal(t, once, RedactSecrets(once), text)
	}
}

func TestRedactSecrets_LeavesWordsThatOnlyStartWithALabel(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"the secretary signs the trading partner agreement",
		"Trading Partner Name: Tokenizer Freight Inc",
		"Acme offers passwordless SFTP with keys",
	} {
		assert.Equal(t, text, RedactSecrets(text))
	}
	assert.Equal(t, "Tokens: [redacted]", RedactSecrets("Tokens: t-1"))
}
