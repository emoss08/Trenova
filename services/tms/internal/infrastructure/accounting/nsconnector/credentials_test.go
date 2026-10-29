package nsconnector

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAccountIDAcceptsNetSuiteAccountForms(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	for raw, want := range map[string]string{
		"1234567":      "1234567",
		" 1234567_sb1": "1234567_SB1",
		"1234567_RP":   "1234567_RP",
		"tstdrv123456": "TSTDRV123456",
	} {
		got, err := conn.NormalizeAccountID(raw)
		require.NoError(t, err, raw)
		assert.Equal(t, want, got)
	}
	_, err := conn.NormalizeAccountID("1234567-s3")
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidAccountID)
}

func TestPrepareCertificateIsValidForTwoYears(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	cert, err := conn.PrepareCertificate("Trenova " + strings.Repeat("Ä", 40))
	require.NoError(t, err)
	assert.Equal(t, testNow.Add(netsuite.MaxCertificateValidity), cert.ExpiresAt)

	block, _ := pem.Decode(cert.CertificatePEM)
	require.NotNil(t, block)
	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(parsed.Subject.CommonName), maxCertificateSubject)
	assert.True(t, strings.HasPrefix(parsed.Subject.CommonName, "Trenova "))
	_, err = netsuite.ParsePrivateKey(cert.PrivateKeyPEM)
	require.NoError(t, err)

	_, err = conn.PrepareCertificate("   ")
	require.ErrorIs(t, err, netsuiteerrors.ErrSubjectRequired)
}

func TestTokenFromCertificateSignsAnAssertionForTheRealmAccount(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPost, tokenPath) {
			return fixtureReply(t, http.StatusOK, "token_client_credentials.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	cert, err := conn.PrepareCertificate("Trenova Acme")
	require.NoError(t, err)

	grant, err := conn.TokenFromCertificate(
		t.Context(),
		&accountingconnectorports.AccountingCertificateGrant{
			RealmID:       testRef().String(),
			CertificateID: "cert-0001",
			PrivateKeyPEM: cert.PrivateKeyPEM,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, &accountingconnectorports.AccountingTokenGrant{
		AccessToken:    "eyJraWQiOiJjLjEyMzQ1NjcifQ.m2m-access.sig",
		AccessTokenTTL: time.Hour,
	}, grant)

	call, ok := fake.find(http.MethodPost, tokenPath)
	require.True(t, ok)
	assert.Equal(t, "client_credentials", call.Form.Get("grant_type"))
	assert.Equal(t, netsuite.ClientAssertionType, call.Form.Get("client_assertion_type"))
	assert.Len(t, strings.Split(call.Form.Get("client_assertion"), "."), 3)
	assert.Empty(t, call.Auth)

	_, err = conn.TokenFromCertificate(
		t.Context(),
		&accountingconnectorports.AccountingCertificateGrant{
			RealmID:       testAccount + "-sX",
			CertificateID: "cert-0001",
			PrivateKeyPEM: cert.PrivateKeyPEM,
		},
	)
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidCompanyRef)
	_, err = conn.TokenFromCertificate(
		t.Context(),
		&accountingconnectorports.AccountingCertificateGrant{
			RealmID:       testAccount,
			CertificateID: "cert-0001",
			PrivateKeyPEM: []byte("not a key"),
		},
	)
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidPrivateKey)
	assert.Equal(t, 1, fake.count(http.MethodPost, tokenPath))
}

func TestTokenFromCertificateRefusedIsRevoked(t *testing.T) {
	t.Parallel()

	for _, fixture := range []string{"invalid_client.json", "invalid_grant.json"} {
		conn := testConnector(t, &fakeNS{respond: func(call nsCall) reply {
			return fixtureReply(t, http.StatusUnauthorized, fixture)
		}})
		cert, err := conn.PrepareCertificate("Trenova Acme")
		require.NoError(t, err)
		_, err = conn.TokenFromCertificate(
			t.Context(),
			&accountingconnectorports.AccountingCertificateGrant{
				RealmID:       testAccount,
				CertificateID: "cert-0001",
				PrivateKeyPEM: cert.PrivateKeyPEM,
			},
		)
		require.ErrorIs(t, err, nsbinding.ErrCertificateRevoked, fixture)
		assert.Equal(t, accountingsync.ErrorCategoryRevoked, conn.ClassifyError(err), fixture)
		assert.Equal(t, accountingsync.SyncErrorAuth, conn.ClassifyDocumentError(err).Category)
	}
}
