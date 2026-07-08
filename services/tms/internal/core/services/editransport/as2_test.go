package editransport

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/stretchr/testify/require"
)

type as2TestIdentity struct {
	certificate    *x509.Certificate
	key            *rsa.PrivateKey
	certificatePEM string
	keyPEM         string
}

func newAS2TestIdentity(t *testing.T, commonName string) *as2TestIdentity {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return &as2TestIdentity{
		certificate: certificate,
		key:         key,
		certificatePEM: string(
			pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		),
		keyPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})),
	}
}

const as2TestPayload = "ISA*00*          *00*          *ZZ*TRENOVA        *ZZ*PARTNER        " +
	"*260107*1200*^*00401*000000001*0*P*>~GS*SM*TRENOVA*PARTNER*20260107*1200*1*X*004010~" +
	"ST*204*0001~B2**SCAC**SHIP1**PP~SE*3*0001~GE*1*1~IEA*1*000000001~"

func as2TestProfile(local, partner *as2TestIdentity, endpointURL, mdnMode string) (
	*edi.EDICommunicationProfile,
	map[string]string,
) {
	profile := &edi.EDICommunicationProfile{
		Method: edi.ConnectionMethodAS2,
		Config: map[string]any{
			ConfigKeyLocalAS2ID:                "TRENOVA-AS2",
			ConfigKeyPartnerAS2ID:              "PARTNER-AS2",
			ConfigKeyEndpointURL:               endpointURL,
			ConfigKeyMDNMode:                   mdnMode,
			ConfigKeyMDNURL:                    "https://trenova.example/api/v1/edi/as2/inbound/",
			ConfigKeyLocalCertificate:          local.certificatePEM,
			ConfigKeyPartnerSigningCertificate: partner.certificatePEM,
		},
	}
	secrets := map[string]string{SecretKeyAS2PrivateKey: local.keyPEM}
	return profile, secrets
}

func TestAS2TransportDeliverSyncMDN(t *testing.T) {
	t.Parallel()

	local := newAS2TestIdentity(t, "trenova.example")
	partner := newAS2TestIdentity(t, "partner.example")

	var receivedPayload []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "TRENOVA-AS2", r.Header.Get("AS2-From"))
		require.Equal(t, "PARTNER-AS2", r.Header.Get("AS2-To"))
		require.NotEmpty(t, r.Header.Get("Message-ID"))
		require.NotEmpty(t, r.Header.Get("Disposition-Notification-To"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		parsed, err := as2.ParseMessage(
			r.Header.Get("Content-Type"),
			body,
			&as2.ParseMessageOptions{
				DecryptionCertificate: partner.certificate,
				DecryptionKey:         partner.key,
				PartnerCertificate:    local.certificate,
				TransferEncoding:      r.Header.Get("Content-Transfer-Encoding"),
				RequireSignature:      true,
				RequireEncryption:     true,
			},
		)
		require.NoError(t, err)
		receivedPayload = parsed.Payload

		mdn, err := as2.BuildMDN(&as2.BuildMDNOptions{
			From:               "PARTNER-AS2",
			To:                 "TRENOVA-AS2",
			OriginalMessageID:  r.Header.Get("Message-ID"),
			ReceivedContentMIC: parsed.MIC,
			SigningCertificate: partner.certificate,
			SigningKey:         partner.key,
		})
		require.NoError(t, err)
		w.Header().Set("Content-Type", mdn.ContentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mdn.Body)
	}))
	defer server.Close()

	profile, secrets := as2TestProfile(local, partner, server.URL, MDNModeSync)
	transport := NewAS2Transport()
	result, err := transport.Deliver(t.Context(), &services.EDITransportRequest{
		Profile:  profile,
		Secrets:  secrets,
		FileName: "shipment.edi",
		Contents: as2TestPayload,
	})
	require.NoError(t, err)
	require.False(t, result.Pending)
	require.NotEmpty(t, result.MessageID)
	require.NotEmpty(t, result.MIC)
	require.Equal(t, as2TestPayload, string(receivedPayload))
}

func TestAS2TransportDeliverAsyncMDNIsPending(t *testing.T) {
	t.Parallel()

	local := newAS2TestIdentity(t, "trenova.example")
	partner := newAS2TestIdentity(t, "partner.example")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotEmpty(t, r.Header.Get("Receipt-Delivery-Option"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	profile, secrets := as2TestProfile(local, partner, server.URL, MDNModeAsync)
	transport := NewAS2Transport()
	result, err := transport.Deliver(t.Context(), &services.EDITransportRequest{
		Profile:  profile,
		Secrets:  secrets,
		FileName: "shipment.edi",
		Contents: as2TestPayload,
	})
	require.NoError(t, err)
	require.True(t, result.Pending)
	require.NotEmpty(t, result.MessageID)
}

func TestAS2TransportDeliverFailsOnRejectedMDN(t *testing.T) {
	t.Parallel()

	local := newAS2TestIdentity(t, "trenova.example")
	partner := newAS2TestIdentity(t, "partner.example")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mdn, err := as2.BuildMDN(&as2.BuildMDNOptions{
			From:              "PARTNER-AS2",
			To:                "TRENOVA-AS2",
			OriginalMessageID: r.Header.Get("Message-ID"),
			ErrorText:         "unknown trading partner",
		})
		require.NoError(t, err)
		w.Header().Set("Content-Type", mdn.ContentType)
		_, _ = w.Write(mdn.Body)
	}))
	defer server.Close()

	profile, secrets := as2TestProfile(local, partner, server.URL, MDNModeSync)
	transport := NewAS2Transport()
	_, err := transport.Deliver(t.Context(), &services.EDITransportRequest{
		Profile:  profile,
		Secrets:  secrets,
		FileName: "shipment.edi",
		Contents: as2TestPayload,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "processing failure")
}

func TestAS2TransportDeliverFailsOnHTTPRejection(t *testing.T) {
	t.Parallel()

	local := newAS2TestIdentity(t, "trenova.example")
	partner := newAS2TestIdentity(t, "partner.example")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	profile, secrets := as2TestProfile(local, partner, server.URL, MDNModeSync)
	transport := NewAS2Transport()
	_, err := transport.Deliver(t.Context(), &services.EDITransportRequest{
		Profile:  profile,
		Secrets:  secrets,
		FileName: "shipment.edi",
		Contents: as2TestPayload,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 403")
}

func TestAS2ConfigFromProfileValidation(t *testing.T) {
	t.Parallel()

	profile := &edi.EDICommunicationProfile{
		Method: edi.ConnectionMethodAS2,
		Config: map[string]any{
			ConfigKeyLocalAS2ID:       "TRENOVA-AS2",
			ConfigKeyPartnerAS2ID:     "PARTNER-AS2",
			ConfigKeyEndpointURL:      "https://partner.example/as2",
			ConfigKeyLocalCertificate: "not a certificate",
		},
	}
	_, err := AS2ConfigFromProfile(profile, map[string]string{})
	require.Error(t, err)

	delete(profile.Config, ConfigKeyLocalCertificate)
	cfg, err := AS2ConfigFromProfile(profile, map[string]string{})
	require.NoError(t, err)
	require.NoError(t, validateAS2DeliveryConfig(cfg))

	cfg.MDNMode = MDNModeAsync
	cfg.MDNURL = ""
	require.Error(t, validateAS2DeliveryConfig(cfg))
}
