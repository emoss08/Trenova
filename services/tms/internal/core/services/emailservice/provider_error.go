package emailservice

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/shared/stringutils"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const providerErrorBodyLimit = 2048

const (
	postmarkErrorSenderSignatureMissing     = 400
	postmarkErrorSenderSignatureUnconfirmed = 401
)

type SenderRejection string

const (
	SenderDomainUnverified     SenderRejection = "DomainUnverified"
	SenderSignatureMissing     SenderRejection = "SignatureMissing"
	SenderSignatureUnconfirmed SenderRejection = "SignatureUnconfirmed"
)

type SenderRejectedError struct {
	Provider  email.Provider
	Rejection SenderRejection
	Address   string
	Origin    string
}

func (e *SenderRejectedError) Error() string {
	address := strings.TrimSpace(e.Address)
	if address == "" {
		address = "the configured sender address"
	}
	domain := stringutils.EmailDomain(address)
	if domain == "" {
		domain = "the sender's domain"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s refused to send from %s because ", e.Provider, address)
	switch e.Rejection {
	case SenderSignatureMissing:
		fmt.Fprintf(
			&b,
			"neither the address nor %s is a confirmed sender in your %s account.",
			domain,
			e.Provider,
		)
	case SenderSignatureUnconfirmed:
		fmt.Fprintf(&b, "its sender signature has not been confirmed in %s.", e.Provider)
	default:
		fmt.Fprintf(&b, "the domain %s is not verified in your %s account.", domain, e.Provider)
	}
	if origin := strings.TrimSpace(e.Origin); origin != "" {
		fmt.Fprintf(&b, " This address comes from %s.", origin)
	}
	fmt.Fprintf(
		&b,
		" Change it to an address on a domain verified in %s, or verify %s in %s.",
		e.Provider,
		domain,
		e.Provider,
	)
	return b.String()
}

func (e *SenderRejectedError) Unwrap() error {
	return serviceports.ErrNonRetryableEmailSend
}

func resendSenderRejection(statusCode int, body []byte) *SenderRejectedError {
	if statusCode != http.StatusForbidden && statusCode != http.StatusUnprocessableEntity {
		return nil
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil
	}
	if !strings.Contains(strings.ToLower(payload.Message), "domain is not verified") {
		return nil
	}
	return &SenderRejectedError{
		Provider:  email.ProviderResend,
		Rejection: SenderDomainUnverified,
	}
}

func postmarkSenderRejection(statusCode int, body []byte) *SenderRejectedError {
	if statusCode != http.StatusUnprocessableEntity {
		return nil
	}
	var payload struct {
		ErrorCode int `json:"ErrorCode"`
	}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil
	}
	switch payload.ErrorCode {
	case postmarkErrorSenderSignatureMissing:
		return &SenderRejectedError{
			Provider:  email.ProviderPostmark,
			Rejection: SenderSignatureMissing,
		}
	case postmarkErrorSenderSignatureUnconfirmed:
		return &SenderRejectedError{
			Provider:  email.ProviderPostmark,
			Rejection: SenderSignatureUnconfirmed,
		}
	default:
		return nil
	}
}

func providerStatusError(base error, provider string, statusCode int, body []byte) error {
	response := strings.TrimSpace(string(body))
	response = strings.Join(strings.Fields(response), " ")
	if len(response) > providerErrorBodyLimit {
		response = response[:providerErrorBodyLimit] + "..."
	}
	if response == "" {
		return fmt.Errorf("%w: %s status %d", base, provider, statusCode)
	}

	return fmt.Errorf("%w: %s status %d: %s", base, provider, statusCode, response)
}

func providerConfigurationError(provider email.Provider, err error) error {
	return fmt.Errorf(
		"%w: %s provider configuration could not be loaded or decrypted; verify integration credentials and encryption key configuration: %w",
		serviceports.ErrNonRetryableEmailSend,
		provider,
		err,
	)
}
