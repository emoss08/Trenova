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
		`the passphrase: "correct horse battery staple" is sent separately`,
		"ISA*AA*guide text\nISA*01*LOGIN     *06*PASSWORD12*ZZ*ACME",
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

func TestRedactSecrets_FindsTheInterchangeAfterTextThatOnlyLooksLikeOne(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"The ISA/GS envelope: ISA*01*XXXXXXXXXX*06*XXXXXXXXXX*ZZ*ACME",
		RedactSecrets("The ISA/GS envelope: ISA*01*LOGIN     *06*PASSWORD12*ZZ*ACME"),
	)
	assert.Equal(t,
		"ISA-level rules a/b/c/d/e/f, then ISA*01*XXXXXXXXXX*06*XXXXXXXXXX*ZZ*ACME",
		RedactSecrets("ISA-level rules a/b/c/d/e/f, then ISA*01*LOGIN     *06*PASSWORD12*ZZ*ACME"),
	)
	assert.Equal(t, "ISA/a/b/c/d/e/f", RedactSecrets("ISA/a/b/c/d/e/f"),
		"text whose qualifiers are not two characters is not an interchange")
}

func TestRedactSecrets_NeverLetsTextThatLooksLikeAnInterchangeSwallowARealOne(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"ISA*AA*guide text\nISA*01*XXXXXXXXXX*06*XXXXXXXXXX*ZZ*ACME",
		RedactSecrets("ISA*AA*guide text\nISA*01*LOGIN     *06*PASSWORD12*ZZ*ACME"),
	)
	assert.Equal(t,
		"ISA*AA*x ISA*01*XXXXXXXXXX*06*XXXXXXXXXX*ZZ*ACME",
		RedactSecrets("ISA*AA*x ISA*01*LOGIN*06*PWD*ZZ*ACME"),
	)
}

func TestRedactSecrets_RedactsAQuotedCredentialWhole(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `the passphrase: [redacted] is sent separately`,
		RedactSecrets(`the passphrase: "correct horse battery staple" is sent separately`))
	assert.Equal(t, "Password: [redacted], then the AS2 ID ACMEAS2",
		RedactSecrets("Password: “alpha beta”, then the AS2 ID ACMEAS2"))
	assert.Equal(t, "Password: [redacted] for the test AS2 ID ACMEAS2",
		RedactSecrets("Password: abc for the test AS2 ID ACMEAS2"),
		"an unquoted value ends at the first space")
}
