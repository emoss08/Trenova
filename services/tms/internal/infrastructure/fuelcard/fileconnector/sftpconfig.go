package fileconnector

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/shared/sftp"
)

// sftpConfigFrom builds the transport settings from an integration's stored
// configuration. The secrets arrive already decrypted, because the integration
// service unwraps every field marked sensitive before handing the map over.
func sftpConfigFrom(config map[string]string) (sftp.Config, error) {
	cfg := sftp.Config{
		Host:         strings.TrimSpace(config[integration.ConfigKeyFuelHost]),
		Port:         strings.TrimSpace(config[integration.ConfigKeyFuelPort]),
		Username:     strings.TrimSpace(config[integration.ConfigKeyFuelUsername]),
		AuthMode:     strings.TrimSpace(config[integration.ConfigKeyFuelAuthMode]),
		KnownHostKey: strings.TrimSpace(config[integration.ConfigKeyFuelKnownHostKey]),
		Password:     strings.TrimSpace(config[integration.ConfigKeyFuelPassword]),
		PrivateKey:   strings.TrimSpace(config[integration.ConfigKeyFuelPrivateKey]),
	}
	if cfg.AuthMode == "" {
		cfg.AuthMode = sftp.AuthModePassword
	}

	if err := cfg.Validate(); err != nil {
		return sftp.Config{}, err
	}

	return cfg, nil
}
