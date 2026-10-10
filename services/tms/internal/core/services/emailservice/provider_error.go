package emailservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
)

const providerErrorBodyLimit = 2048

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
		emailports.ErrNonRetryableEmailSend,
		provider,
		err,
	)
}
