package editransport

import (
	"testing"

	"github.com/emoss08/trenova/shared/as2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	syncMDNMessageID = "<sent@trenova.as2>"
	syncMDNMIC       = "q1w2e3r4, sha256"
)

func sentSyncMessage() *as2.BuiltMessage {
	return &as2.BuiltMessage{MessageID: syncMDNMessageID, MIC: syncMDNMIC}
}

func buildSyncMDN(t *testing.T, opts as2.BuildMDNOptions) *as2.BuiltMDN {
	t.Helper()

	opts.From = "PARTNER-AS2"
	opts.To = "TRENOVA-AS2"
	mdn, err := as2.BuildMDN(&opts)
	require.NoError(t, err)
	return mdn
}

func TestSyncMDNMustBeSignedWhenThePartnerHasASigningCertificate(t *testing.T) {
	t.Parallel()

	partner := newAS2TestIdentity(t, "partner.example")
	mdn := buildSyncMDN(t, as2.BuildMDNOptions{
		OriginalMessageID:  syncMDNMessageID,
		ReceivedContentMIC: syncMDNMIC,
	})

	err := verifySyncMDN(
		mdn.ContentType,
		mdn.Body,
		&AS2Config{PartnerSigningCertificate: partner.certificate},
		sentSyncMessage(),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be signed")
}

func TestSyncMDNSignedByAnotherCertificateIsRefused(t *testing.T) {
	t.Parallel()

	partner := newAS2TestIdentity(t, "partner.example")
	stranger := newAS2TestIdentity(t, "stranger.example")
	mdn := buildSyncMDN(t, as2.BuildMDNOptions{
		OriginalMessageID:  syncMDNMessageID,
		ReceivedContentMIC: syncMDNMIC,
		SigningCertificate: stranger.certificate,
		SigningKey:         stranger.key,
	})

	err := verifySyncMDN(
		mdn.ContentType,
		mdn.Body,
		&AS2Config{PartnerSigningCertificate: partner.certificate},
		sentSyncMessage(),
	)

	require.Error(t, err)
}

func TestSignedSyncMDNIsAcceptedWithoutAMIC(t *testing.T) {
	t.Parallel()

	partner := newAS2TestIdentity(t, "partner.example")
	mdn := buildSyncMDN(t, as2.BuildMDNOptions{
		OriginalMessageID:  syncMDNMessageID,
		SigningCertificate: partner.certificate,
		SigningKey:         partner.key,
	})

	require.NoError(t, verifySyncMDN(
		mdn.ContentType,
		mdn.Body,
		&AS2Config{PartnerSigningCertificate: partner.certificate},
		sentSyncMessage(),
	))
}

func TestSignedSyncMDNWithAnotherMICIsRefused(t *testing.T) {
	t.Parallel()

	partner := newAS2TestIdentity(t, "partner.example")
	mdn := buildSyncMDN(t, as2.BuildMDNOptions{
		OriginalMessageID:  syncMDNMessageID,
		ReceivedContentMIC: "other, sha256",
		SigningCertificate: partner.certificate,
		SigningKey:         partner.key,
	})

	err := verifySyncMDN(
		mdn.ContentType,
		mdn.Body,
		&AS2Config{PartnerSigningCertificate: partner.certificate},
		sentSyncMessage(),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "MIC does not match")
}

func TestUnsignedSyncMDNMustCarryTheTransmittedMIC(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		mic     string
		message string
	}{
		{name: "absent", message: "could not be authenticated"},
		{name: "different", mic: "other, sha256", message: "MIC does not match"},
		{name: "matching", mic: syncMDNMIC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mdn := buildSyncMDN(t, as2.BuildMDNOptions{
				OriginalMessageID:  syncMDNMessageID,
				ReceivedContentMIC: tc.mic,
			})

			err := verifySyncMDN(mdn.ContentType, mdn.Body, &AS2Config{}, sentSyncMessage())

			if tc.message == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.message)
		})
	}
}

func TestSyncMDNForAnotherMessageIsRefused(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		messageID string
	}{
		{name: "another message", messageID: "<replayed@trenova.as2>"},
		{name: "no reference"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mdn := buildSyncMDN(t, as2.BuildMDNOptions{
				OriginalMessageID:  tc.messageID,
				ReceivedContentMIC: syncMDNMIC,
			})

			err := verifySyncMDN(mdn.ContentType, mdn.Body, &AS2Config{}, sentSyncMessage())

			require.Error(t, err)
			assert.Contains(t, err.Error(), "does not reference the transmitted message")
		})
	}
}
