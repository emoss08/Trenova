package connection

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/envconfig"
	"go.uber.org/zap"
)

const (
	sourceConfig  = "config"
	sourceProfile = "profile"

	authNone   = "none"
	authAPIKey = "api-key"
	authMTLS   = "mtls"
)

var (
	ErrTemporalAddressRequired = errors.New(
		"temporal address is required: set temporal.hostPort or an address in the client profile",
	)
	ErrTemporalAPIKeyRequiresTLS = errors.New(
		"temporal api key cannot be sent over a plaintext connection: enable TLS",
	)
)

type Summary struct {
	Source    string
	Profile   string
	HostPort  string
	Namespace string
	Auth      string
	TLS       bool
}

func BuildOptions(
	cfg *config.TemporalConfig,
) (client.Options, Summary, error) {
	profile, summary, err := resolveClientProfile(cfg)
	if err != nil {
		return client.Options{}, Summary{}, err
	}

	if profile.APIKey != "" && profile.TLS != nil && profile.TLS.Disabled {
		return client.Options{}, Summary{}, ErrTemporalAPIKeyRequiresTLS
	}

	opts, err := profile.ToClientOptions(envconfig.ToClientOptionsRequest{})
	if err != nil {
		return client.Options{}, Summary{}, fmt.Errorf(
			"build temporal client options: %w",
			err,
		)
	}

	opts.HostPort = strings.TrimSpace(opts.HostPort)
	if opts.HostPort == "" {
		return client.Options{}, Summary{}, ErrTemporalAddressRequired
	}

	if strings.TrimSpace(opts.Namespace) == "" {
		opts.Namespace = cfg.GetNamespace()
	}
	opts.Identity = cfg.GetIdentity()

	summary.HostPort = opts.HostPort
	summary.Namespace = opts.Namespace
	summary.TLS = opts.ConnectionOptions.TLS != nil
	summary.Auth = authMode(&profile)

	return opts, summary, nil
}

func resolveClientProfile(
	cfg *config.TemporalConfig,
) (envconfig.ClientConfigProfile, Summary, error) {
	if cfg.UsesProfile() {
		name := strings.TrimSpace(cfg.Profile)
		profile, err := envconfig.LoadClientConfigProfile(envconfig.LoadClientConfigProfileOptions{
			ConfigFilePath:    strings.TrimSpace(cfg.ConfigFile),
			ConfigFileProfile: name,
		})
		if err != nil {
			return envconfig.ClientConfigProfile{}, Summary{}, fmt.Errorf(
				"load temporal client profile %q: %w",
				name,
				err,
			)
		}

		return profile, Summary{Source: sourceProfile, Profile: name}, nil
	}

	profile := envconfig.ClientConfigProfile{
		Address:   strings.TrimSpace(cfg.HostPort),
		Namespace: cfg.GetNamespace(),
		APIKey:    strings.TrimSpace(cfg.APIKey),
	}

	if cfg.TLS.IsEnabled() {
		profile.TLS = &envconfig.ClientConfigTLS{
			ServerName:       cfg.TLS.ServerName,
			ServerCACertPath: cfg.TLS.ServerCACertPath,
			ClientCertPath:   cfg.TLS.ClientCertPath,
			ClientKeyPath:    cfg.TLS.ClientKeyPath,
		}
	}

	return profile, Summary{Source: sourceConfig}, nil
}

func authMode(profile *envconfig.ClientConfigProfile) string {
	switch {
	case profile.APIKey != "":
		return authAPIKey
	case profile.TLS != nil &&
		(profile.TLS.ClientCertPath != "" || len(profile.TLS.ClientCertData) > 0):
		return authMTLS
	default:
		return authNone
	}
}

func (s Summary) Fields() []zap.Field {
	fields := make([]zap.Field, 0, 6)
	fields = append(fields,
		zap.String("source", s.Source),
		zap.String("hostPort", s.HostPort),
		zap.String("namespace", s.Namespace),
		zap.String("auth", s.Auth),
		zap.Bool("tls", s.TLS),
	)
	if s.Profile != "" {
		fields = append(fields, zap.String("profile", s.Profile))
	}

	return fields
}
