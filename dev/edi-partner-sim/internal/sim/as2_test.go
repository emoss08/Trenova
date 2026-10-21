package sim

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/as2"
)

const interopPayload = "ISA*00*          *00*          *ZZ*TRENOVA        *ZZ*PARTNER        " +
	"*260107*1200*^*00401*000000001*0*P*>~GS*SM*TRENOVA*PARTNER*20260107*1200*1*X*004010~" +
	"ST*204*0001~B2**SCAC**SHIP1**PP~SE*3*0001~GE*1*1~IEA*1*000000001~"

func newInteropSender(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "trenova.interop"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return certificate, key
}

func TestPartnerSimulatorDecryptsEveryEncryptionAlgorithm(t *testing.T) {
	t.Parallel()

	server, err := NewServer(Options{
		AS2ID:       "PARTNER",
		RemoteAS2ID: "TRENOVA",
		IdentityDir: t.TempDir(),
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	senderCert, senderKey := newInteropSender(t)
	server.partnerCert = senderCert

	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	algorithms := []string{
		as2.EncryptionAlgorithmTripleDES,
		as2.EncryptionAlgorithmAES128CBC,
		as2.EncryptionAlgorithmAES256CBC,
		as2.EncryptionAlgorithmAES128GCM,
		as2.EncryptionAlgorithmAES256GCM,
	}
	for _, algorithm := range algorithms {
		built, buildErr := as2.BuildMessage(&as2.BuildMessageOptions{
			From:                  "TRENOVA",
			To:                    "PARTNER",
			FileName:              "tender.edi",
			Payload:               []byte(interopPayload),
			SigningCertificate:    senderCert,
			SigningKey:            senderKey,
			EncryptionCertificate: server.identity.Certificate,
			EncryptionAlgorithm:   algorithm,
		})
		if buildErr != nil {
			t.Fatalf("%s: build message: %v", algorithm, buildErr)
		}
		request, requestErr := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			httpServer.URL+"/as2",
			bytes.NewReader(built.Body),
		)
		if requestErr != nil {
			t.Fatalf("%s: new request: %v", algorithm, requestErr)
		}
		for key, values := range built.Headers {
			for _, value := range values {
				request.Header.Add(key, value)
			}
		}
		request.Header.Set("Content-Type", built.ContentType)
		response, doErr := http.DefaultClient.Do(request)
		if doErr != nil {
			t.Fatalf("%s: post: %v", algorithm, doErr)
		}
		_ = response.Body.Close()
	}

	server.mu.RLock()
	received := append([]*ReceivedDocument(nil), server.received...)
	server.mu.RUnlock()
	if len(received) != len(algorithms) {
		t.Fatalf("expected %d documents, got %d", len(algorithms), len(received))
	}
	for i, document := range received {
		if document.RejectedReason != "" {
			t.Fatalf("%s: rejected: %s", algorithms[i], document.RejectedReason)
		}
		if !document.Encrypted || !document.Signed {
			t.Fatalf("%s: expected a signed, encrypted document", algorithms[i])
		}
		if document.Payload != interopPayload {
			t.Fatalf("%s: payload did not round trip", algorithms[i])
		}
	}
}
