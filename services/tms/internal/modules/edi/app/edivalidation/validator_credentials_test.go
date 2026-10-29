package edivalidation

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sftpProfileWithoutCredentials(status domaintypes.Status) *edi.EDICommunicationProfile {
	return &edi.EDICommunicationProfile{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		EDIPartnerID:   pulid.MustNew("edip_"),
		Method:         edi.ConnectionMethodSFTP,
		Status:         status,
		Name:           "Acme Retail SFTP",
		Config: map[string]any{
			"host":                 "sftp.acmeretail.example",
			"port":                 "22",
			"username":             "trenova",
			"authMode":             "password",
			"isaSenderQualifier":   "ZZ",
			"isaSenderId":          "TRENOVA",
			"isaReceiverQualifier": "ZZ",
			"isaReceiverId":        "ACMERETAIL",
			"gsSenderId":           "TRENOVA",
			"gsReceiverId":         "ACMERETAIL",
			"x12Version":           "004010",
			"environment":          "test",
		},
	}
}

func multiErrFields(multiErr *errortypes.MultiError) []string {
	if multiErr == nil {
		return nil
	}
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestValidateCommunicationProfile_AnInactiveProfileMayWaitForItsCredentials(t *testing.T) {
	t.Parallel()

	assert.Nil(t, NewValidator().ValidateCommunicationProfile(
		sftpProfileWithoutCredentials(domaintypes.StatusInactive),
	))
}

func TestValidateCommunicationProfile_AnActiveProfileNeedsItsCredentials(t *testing.T) {
	t.Parallel()

	multiErr := NewValidator().ValidateCommunicationProfile(
		sftpProfileWithoutCredentials(domaintypes.StatusActive),
	)
	require.NotNil(t, multiErr)
	assert.ElementsMatch(t, []string{"config.knownHostKey", "secrets"}, multiErrFields(multiErr))
}

func TestValidateCommunicationProfile_AnInactiveProfileStillNeedsItsSettings(t *testing.T) {
	t.Parallel()

	profile := sftpProfileWithoutCredentials(domaintypes.StatusInactive)
	delete(profile.Config, "host")

	multiErr := NewValidator().ValidateCommunicationProfile(profile)
	require.NotNil(t, multiErr)
	assert.Equal(t, []string{"config.host"}, multiErrFields(multiErr))
}

func TestValidateCommunicationProfile_AnInactiveAS2ProfileMayWaitForItsPrivateKey(t *testing.T) {
	t.Parallel()

	profile := sftpProfileWithoutCredentials(domaintypes.StatusInactive)
	profile.Method = edi.ConnectionMethodAS2
	profile.Config["localAS2Id"] = "TRENOVA"
	profile.Config["partnerAS2Id"] = "ACMERETAIL"
	profile.Config["endpointUrl"] = "https://as2.acmeretail.example/as2"
	profile.Config["mdnMode"] = "sync"
	profile.Config["localCertificate"] = testCertificatePEM(t)

	assert.Nil(t, NewValidator().ValidateCommunicationProfile(profile))

	profile.Status = domaintypes.StatusActive
	assert.Equal(t, []string{"secrets"},
		multiErrFields(NewValidator().ValidateCommunicationProfile(profile)))
}

func TestCredentialRequirements_NamesWhatAPersonMustEnter(t *testing.T) {
	t.Parallel()

	sftp := sftpProfileWithoutCredentials(domaintypes.StatusInactive)
	assert.Equal(t, []ediconnectiondraftports.CredentialRequirement{
		{
			Keys:     []string{"knownHostKey"},
			Label:    "Known host key",
			Kind:     ediconnectiondraftports.CredentialKindSetting,
			Required: true,
			Missing:  true,
		},
		{
			Keys:     []string{"password", "privateKey"},
			Label:    "SFTP password or private key",
			Kind:     ediconnectiondraftports.CredentialKindSecret,
			Required: true,
			Missing:  true,
		},
	}, missingOnly(CredentialRequirements(sftp)))

	sftp.Config["knownHostKey"] = "ssh-ed25519 AAAAC3Nza"
	sftp.EncryptedSecrets = map[string]string{"privateKey": "trenova-envelope:v1:abc"}
	assert.Empty(t, missingOnly(CredentialRequirements(sftp)))
}

func missingOnly(
	requirements []ediconnectiondraftports.CredentialRequirement,
) []ediconnectiondraftports.CredentialRequirement {
	out := make([]ediconnectiondraftports.CredentialRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		if requirement.Missing {
			out = append(out, requirement)
		}
	}

	return out
}

func testCertificatePEM(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "TRENOVA"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
