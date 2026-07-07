package editransport

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/maputils"
)

const (
	ConfigKeyLocalAS2ID                   = "localAS2Id"
	ConfigKeyPartnerAS2ID                 = "partnerAS2Id"
	ConfigKeyEndpointURL                  = "endpointUrl"
	ConfigKeyMDNMode                      = "mdnMode"
	ConfigKeyMDNURL                       = "mdnUrl"
	ConfigKeySigningAlgorithm             = "signingAlgorithm"
	ConfigKeyEncryptionAlgorithm          = "encryptionAlgorithm"
	ConfigKeyCompressionAlgorithm         = "compressionAlgorithm"
	ConfigKeyLocalCertificate             = "localCertificate"
	ConfigKeyPartnerSigningCertificate    = "partnerSigningCertificate"
	ConfigKeyPartnerEncryptionCertificate = "partnerEncryptionCertificate"
	ConfigKeyBasicAuthUsername            = "basicAuthUsername"

	SecretKeyAS2PrivateKey     = "privateKey"
	SecretKeyBasicAuthPassword = "basicAuthPassword"

	MDNModeSync  = "sync"
	MDNModeAsync = "async"

	CompressionZlib = "zlib"

	as2RequestTimeout = 60 * time.Second
	as2MaxMDNBody     = 1 << 20
)

type AS2Config struct {
	LocalAS2ID                   string
	PartnerAS2ID                 string
	EndpointURL                  string
	MDNMode                      string
	MDNURL                       string
	SigningAlgorithm             string
	EncryptionAlgorithm          string
	CompressionAlgorithm         string
	BasicAuthUsername            string
	BasicAuthPassword            string
	LocalCertificate             *x509.Certificate
	PrivateKey                   crypto.PrivateKey
	PartnerSigningCertificate    *x509.Certificate
	PartnerEncryptionCertificate *x509.Certificate
}

func (c *AS2Config) Async() bool {
	return strings.EqualFold(c.MDNMode, MDNModeAsync)
}

func AS2ConfigFromProfile(
	profile *edi.EDICommunicationProfile,
	secrets map[string]string,
) (*AS2Config, error) {
	cfg := &AS2Config{
		LocalAS2ID:           maputils.StringValue(profile.Config, ConfigKeyLocalAS2ID),
		PartnerAS2ID:         maputils.StringValue(profile.Config, ConfigKeyPartnerAS2ID),
		EndpointURL:          maputils.StringValue(profile.Config, ConfigKeyEndpointURL),
		MDNMode:              maputils.StringValue(profile.Config, ConfigKeyMDNMode),
		MDNURL:               maputils.StringValue(profile.Config, ConfigKeyMDNURL),
		SigningAlgorithm:     maputils.StringValue(profile.Config, ConfigKeySigningAlgorithm),
		EncryptionAlgorithm:  maputils.StringValue(profile.Config, ConfigKeyEncryptionAlgorithm),
		CompressionAlgorithm: maputils.StringValue(profile.Config, ConfigKeyCompressionAlgorithm),
		BasicAuthUsername:    maputils.StringValue(profile.Config, ConfigKeyBasicAuthUsername),
		BasicAuthPassword:    strings.TrimSpace(secrets[SecretKeyBasicAuthPassword]),
	}
	if pemData := maputils.StringValue(profile.Config, ConfigKeyLocalCertificate); pemData != "" {
		certificate, err := as2.ParseCertificate([]byte(pemData))
		if err != nil {
			return nil, fmt.Errorf("local AS2 certificate is invalid: %w", err)
		}
		cfg.LocalCertificate = certificate
	}
	if pemData := maputils.StringValue(
		profile.Config,
		ConfigKeyPartnerSigningCertificate,
	); pemData != "" {
		certificate, err := as2.ParseCertificate([]byte(pemData))
		if err != nil {
			return nil, fmt.Errorf("partner AS2 signing certificate is invalid: %w", err)
		}
		cfg.PartnerSigningCertificate = certificate
	}
	if pemData := maputils.StringValue(
		profile.Config,
		ConfigKeyPartnerEncryptionCertificate,
	); pemData != "" {
		certificate, err := as2.ParseCertificate([]byte(pemData))
		if err != nil {
			return nil, fmt.Errorf("partner AS2 encryption certificate is invalid: %w", err)
		}
		cfg.PartnerEncryptionCertificate = certificate
	}
	if cfg.PartnerEncryptionCertificate == nil {
		cfg.PartnerEncryptionCertificate = cfg.PartnerSigningCertificate
	}
	if keyPEM := strings.TrimSpace(secrets[SecretKeyAS2PrivateKey]); keyPEM != "" {
		key, err := as2.ParsePrivateKey([]byte(keyPEM))
		if err != nil {
			return nil, fmt.Errorf("AS2 private key secret is invalid: %w", err)
		}
		cfg.PrivateKey = key
	}
	return cfg, nil
}

type AS2Transport struct {
	client *http.Client
}

func NewAS2Transport() *AS2Transport {
	return &AS2Transport{client: &http.Client{Timeout: as2RequestTimeout}}
}

func (t *AS2Transport) Method() edi.ConnectionMethod {
	return edi.ConnectionMethodAS2
}

func (t *AS2Transport) Deliver(
	ctx context.Context,
	req *services.EDITransportRequest,
) (*services.EDITransportResult, error) {
	if req == nil || req.Profile == nil {
		return nil, ErrEDICommunicationProfileRequired
	}
	cfg, err := AS2ConfigFromProfile(req.Profile, req.Secrets)
	if err != nil {
		return nil, err
	}
	if err = validateAS2DeliveryConfig(cfg); err != nil {
		return nil, err
	}

	built, err := as2.BuildMessage(&as2.BuildMessageOptions{
		From:                  cfg.LocalAS2ID,
		To:                    cfg.PartnerAS2ID,
		Subject:               "Trenova EDI Document",
		FileName:              req.FileName,
		Payload:               []byte(req.Contents),
		SigningCertificate:    cfg.LocalCertificate,
		SigningKey:            cfg.PrivateKey,
		EncryptionCertificate: cfg.PartnerEncryptionCertificate,
		SigningAlgorithm:      cfg.SigningAlgorithm,
		EncryptionAlgorithm:   cfg.EncryptionAlgorithm,
		MICAlgorithm:          cfg.SigningAlgorithm,
		Compress:              strings.EqualFold(cfg.CompressionAlgorithm, CompressionZlib),
		RequestMDN:            true,
		RequestSignedMDN:      cfg.PartnerSigningCertificate != nil,
		AsyncMDNURL:           asyncMDNURL(cfg),
	})
	if err != nil {
		return nil, err
	}

	response, err := t.post(ctx, cfg, built)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, as2MaxMDNBody))
	if err != nil {
		return nil, fmt.Errorf("AS2 partner response could not be read: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf(
			"AS2 partner rejected the transmission with status %d: %s",
			response.StatusCode,
			truncateForError(responseBody),
		)
	}

	result := &services.EDITransportResult{
		RemotePath: cfg.EndpointURL,
		MessageID:  built.MessageID,
		MIC:        built.MIC,
	}
	if cfg.Async() {
		result.Pending = true
		return result, nil
	}
	if err = verifySyncMDN(
		response.Header.Get("Content-Type"),
		responseBody,
		cfg,
		built.MIC,
	); err != nil {
		return nil, err
	}
	return result, nil
}

func (t *AS2Transport) post(
	ctx context.Context,
	cfg *AS2Config,
	built *as2.BuiltMessage,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		cfg.EndpointURL,
		bytes.NewReader(built.Body),
	)
	if err != nil {
		return nil, fmt.Errorf("AS2 request could not be created: %w", err)
	}
	request.Header.Set("Content-Type", built.ContentType)
	for key, values := range built.Headers {
		for _, value := range values {
			request.Header.Set(key, value)
		}
	}
	if cfg.BasicAuthUsername != "" {
		request.SetBasicAuth(cfg.BasicAuthUsername, cfg.BasicAuthPassword)
	}
	response, err := t.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("AS2 transmission to %s failed: %w", cfg.EndpointURL, err)
	}
	return response, nil
}

func verifySyncMDN(contentType string, body []byte, cfg *AS2Config, sentMIC string) error {
	if strings.TrimSpace(contentType) == "" || len(bytes.TrimSpace(body)) == 0 {
		return errors.New(
			"AS2 partner did not return a synchronous MDN; configure async MDN mode for this partner",
		)
	}
	mdn, err := as2.ParseMDN(contentType, body, cfg.PartnerSigningCertificate)
	if err != nil {
		return fmt.Errorf("AS2 synchronous MDN could not be parsed: %w", err)
	}
	if !mdn.Processed() {
		return fmt.Errorf(
			"AS2 partner reported a processing failure: %s",
			mdn.Disposition,
		)
	}
	if mdn.ReceivedContentMIC != "" && !as2.MICMatches(sentMIC, mdn.ReceivedContentMIC) {
		return errors.New("AS2 MDN MIC does not match the transmitted content")
	}

	return nil
}

func validateAS2DeliveryConfig(cfg *AS2Config) error {
	switch {
	case cfg.LocalAS2ID == "":
		return errors.New("local AS2 ID is required for AS2 delivery")
	case cfg.PartnerAS2ID == "":
		return errors.New("partner AS2 ID is required for AS2 delivery")
	case cfg.EndpointURL == "":
		return errors.New("partner endpoint URL is required for AS2 delivery")
	case cfg.LocalCertificate != nil && cfg.PrivateKey == nil:
		return errors.New(
			"AS2 private key secret is required when a local certificate is configured",
		)
	case cfg.LocalCertificate == nil && cfg.PrivateKey != nil:
		return errors.New("local AS2 certificate is required when a private key is configured")
	case cfg.Async() && cfg.MDNURL == "":
		return errors.New("async MDN return URL is required when MDN mode is async")
	default:
		return nil
	}
}

func asyncMDNURL(cfg *AS2Config) string {
	if cfg.Async() {
		return cfg.MDNURL
	}
	return ""
}

func truncateForError(body []byte) string {
	text := strings.TrimSpace(string(body))
	if len(text) > 256 {
		return text[:256] + "..."
	}
	return text
}
