package as2

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"strings"
	"time"
)

type CertificateSummary struct {
	Subject           string `json:"subject"`
	Issuer            string `json:"issuer"`
	SerialNumber      string `json:"serialNumber"`
	NotBefore         int64  `json:"notBefore"`
	NotAfter          int64  `json:"notAfter"`
	ExpiresInDays     int    `json:"expiresInDays"`
	Expired           bool   `json:"expired"`
	SHA256Fingerprint string `json:"sha256Fingerprint"`
}

func SummarizeCertificate(cert *x509.Certificate) CertificateSummary {
	digest := sha256.Sum256(cert.Raw)
	encoded := strings.ToUpper(hex.EncodeToString(digest[:]))
	pairs := make([]string, 0, len(encoded)/2)
	for index := 0; index < len(encoded); index += 2 {
		pairs = append(pairs, encoded[index:index+2])
	}
	now := time.Now()
	return CertificateSummary{
		Subject:           cert.Subject.String(),
		Issuer:            cert.Issuer.String(),
		SerialNumber:      cert.SerialNumber.String(),
		NotBefore:         cert.NotBefore.Unix(),
		NotAfter:          cert.NotAfter.Unix(),
		ExpiresInDays:     int(cert.NotAfter.Sub(now).Hours() / 24),
		Expired:           now.After(cert.NotAfter),
		SHA256Fingerprint: strings.Join(pairs, ":"),
	}
}
