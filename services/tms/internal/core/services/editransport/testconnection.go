package editransport

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/maputils"
)

const (
	certificateExpiryWarningWindow = 30 * 24 * time.Hour
	as2ReachabilityTimeout         = 15 * time.Second
)

func passedCheck(name, message string) services.EDIConnectionCheck {
	return services.EDIConnectionCheck{
		Name:    name,
		Status:  services.EDIConnectionCheckPassed,
		Message: message,
	}
}

func warningCheck(name, message string) services.EDIConnectionCheck {
	return services.EDIConnectionCheck{
		Name:    name,
		Status:  services.EDIConnectionCheckWarning,
		Message: message,
	}
}

func failedCheck(name, message string) services.EDIConnectionCheck {
	return services.EDIConnectionCheck{
		Name:    name,
		Status:  services.EDIConnectionCheckFailed,
		Message: message,
	}
}

func (t *SFTPTransport) TestConnection(
	ctx context.Context,
	req *services.EDITransportRequest,
) []services.EDIConnectionCheck {
	return testSFTPEndpoint(ctx, req, defaultOutboundDirectory)
}

func (t *VANTransport) TestConnection(
	ctx context.Context,
	req *services.EDITransportRequest,
) []services.EDIConnectionCheck {
	if req == nil || req.Profile == nil {
		return []services.EDIConnectionCheck{
			failedCheck("configuration", "EDI communication profile is required"),
		}
	}
	mailboxID := stringOrDefault(
		maputils.StringValue(req.Profile.Config, configKeyVANMailboxID),
		"",
	)
	if mailboxID == "" {
		return []services.EDIConnectionCheck{
			failedCheck("configuration", "VAN mailbox ID is required"),
		}
	}
	return testSFTPEndpoint(ctx, req, path.Join("/", mailboxID, "outbound"))
}

func testSFTPEndpoint(
	ctx context.Context,
	req *services.EDITransportRequest,
	fallbackOutboundDirectory string,
) []services.EDIConnectionCheck {
	checks := make([]services.EDIConnectionCheck, 0, 4)
	if req == nil || req.Profile == nil {
		return append(checks, failedCheck("configuration", "EDI communication profile is required"))
	}
	cfg := endpointConfigFromProfile(req.Profile, req.Secrets)
	if err := validateEndpointConfig(&cfg); err != nil {
		return append(checks, failedCheck("configuration", err.Error()))
	}
	checks = append(checks, passedCheck("configuration", "Endpoint configuration is complete"))

	client, sshClient, err := dialSFTP(ctx, &cfg)
	if err != nil {
		return append(checks, failedCheck("connection", err.Error()))
	}
	defer sshClient.Close()
	defer client.Close()
	checks = append(checks, passedCheck("connection", "Connected and authenticated"))

	outboundDirectory := stringOrDefault(cfg.outboundDirectory, fallbackOutboundDirectory)
	if _, statErr := client.Stat(outboundDirectory); statErr != nil {
		checks = append(checks, warningCheck(
			"outbound directory",
			fmt.Sprintf("%s is not accessible: %s", outboundDirectory, statErr.Error()),
		))
	} else {
		checks = append(checks, passedCheck(
			"outbound directory",
			outboundDirectory+" is accessible",
		))
	}

	if inbound := maputils.StringValue(req.Profile.Config, "inboundDirectory"); inbound != "" {
		if _, statErr := client.Stat(inbound); statErr != nil {
			checks = append(checks, warningCheck(
				"inbound directory",
				fmt.Sprintf("%s is not accessible: %s", inbound, statErr.Error()),
			))
		} else {
			checks = append(checks, passedCheck("inbound directory", inbound+" is accessible"))
		}
	}
	return checks
}

func (t *AS2Transport) TestConnection(
	ctx context.Context,
	req *services.EDITransportRequest,
) []services.EDIConnectionCheck {
	checks := make([]services.EDIConnectionCheck, 0, 5)
	if req == nil || req.Profile == nil {
		return append(checks, failedCheck("configuration", "EDI communication profile is required"))
	}
	cfg, err := AS2ConfigFromProfile(req.Profile, req.Secrets)
	if err != nil {
		return append(checks, failedCheck("configuration", err.Error()))
	}
	if validationErr := validateAS2DeliveryConfig(cfg); validationErr != nil {
		checks = append(checks, failedCheck("configuration", validationErr.Error()))
	} else {
		checks = append(checks, passedCheck("configuration", "AS2 configuration is complete"))
	}

	checks = append(checks, testAS2LocalIdentity(cfg))
	if cfg.PartnerSigningCertificate != nil {
		checks = append(checks, certificateExpiryCheck(
			"partner signing certificate",
			cfg.PartnerSigningCertificate,
		))
	}
	if cfg.PartnerEncryptionCertificate != nil &&
		cfg.PartnerEncryptionCertificate != cfg.PartnerSigningCertificate {
		checks = append(checks, certificateExpiryCheck(
			"partner encryption certificate",
			cfg.PartnerEncryptionCertificate,
		))
	}
	checks = append(checks, testAS2Reachability(ctx, t.client, cfg))
	return checks
}

func testAS2LocalIdentity(cfg *AS2Config) services.EDIConnectionCheck {
	if cfg.LocalCertificate == nil {
		return failedCheck("local identity", "Local certificate is not configured")
	}
	if cfg.PrivateKey == nil {
		return failedCheck("local identity", "AS2 private key secret is not configured")
	}
	signer, ok := cfg.PrivateKey.(crypto.Signer)
	if !ok {
		return failedCheck("local identity", "AS2 private key type is not supported")
	}
	publicKey, ok := signer.Public().(interface{ Equal(x crypto.PublicKey) bool })
	if !ok || !publicKey.Equal(cfg.LocalCertificate.PublicKey) {
		return failedCheck(
			"local identity",
			"AS2 private key does not match the local certificate",
		)
	}
	expiry := certificateExpiryCheck("local certificate", cfg.LocalCertificate)
	if expiry.Status != services.EDIConnectionCheckPassed {
		return expiry
	}
	return passedCheck("local identity", "Private key matches the local certificate")
}

func certificateExpiryCheck(name string, cert *x509.Certificate) services.EDIConnectionCheck {
	now := time.Now()
	switch {
	case now.After(cert.NotAfter):
		return failedCheck(name, fmt.Sprintf(
			"Certificate expired on %s",
			cert.NotAfter.UTC().Format(time.RFC3339),
		))
	case now.Before(cert.NotBefore):
		return failedCheck(name, fmt.Sprintf(
			"Certificate is not valid until %s",
			cert.NotBefore.UTC().Format(time.RFC3339),
		))
	case now.Add(certificateExpiryWarningWindow).After(cert.NotAfter):
		return warningCheck(name, fmt.Sprintf(
			"Certificate expires in %d day(s)",
			int(time.Until(cert.NotAfter).Hours()/24),
		))
	default:
		return passedCheck(name, fmt.Sprintf(
			"Certificate valid until %s",
			cert.NotAfter.UTC().Format("2006-01-02"),
		))
	}
}

func testAS2Reachability(
	ctx context.Context,
	client *http.Client,
	cfg *AS2Config,
) services.EDIConnectionCheck {
	if cfg.EndpointURL == "" {
		return failedCheck("endpoint", "Endpoint URL is not configured")
	}
	requestCtx, cancel := context.WithTimeout(ctx, as2ReachabilityTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodHead,
		cfg.EndpointURL,
		nil,
	)
	if err != nil {
		return failedCheck("endpoint", "Endpoint URL is invalid: "+err.Error())
	}
	if cfg.BasicAuthUsername != "" {
		request.SetBasicAuth(cfg.BasicAuthUsername, cfg.BasicAuthPassword)
	}
	response, err := client.Do(request)
	if err != nil {
		return failedCheck("endpoint", "Endpoint is not reachable: "+err.Error())
	}
	defer response.Body.Close()
	return passedCheck("endpoint", fmt.Sprintf(
		"Endpoint responded with HTTP %d",
		response.StatusCode,
	))
}
