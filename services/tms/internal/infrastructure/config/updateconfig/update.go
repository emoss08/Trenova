package updateconfig

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	Enabled         bool          `mapstructure:"enabled"`
	CheckInterval   time.Duration `mapstructure:"checkInterval"`
	GitHubOwner     string        `mapstructure:"githubOwner"`
	GitHubRepo      string        `mapstructure:"githubRepo"`
	AllowPrerelease bool          `mapstructure:"allowPrerelease"`
	ProxyURL        string        `mapstructure:"proxyUrl"`
	OfflineMode     bool          `mapstructure:"offlineMode"`
	// CaptureManifestURL is where the signed Trenova Capture release manifest
	// is read from. It defaults to the rolling capture-stable release of the
	// configured repository.
	CaptureManifestURL string `mapstructure:"captureManifestUrl" validate:"omitempty,url"`
	// CapturePublicKey is the base64 ed25519 key Trenova Capture releases are
	// signed with. Without it no release is served.
	CapturePublicKey string `mapstructure:"capturePublicKey"`
}

const (
	captureStableTag     = "capture-stable"
	captureManifestAsset = "trenova-capture-manifest.json"
)

// GetCaptureManifestURL is the configured manifest address, or the
// repository's capture-stable release asset.
func (c *Config) GetCaptureManifestURL() string {
	if configured := strings.TrimSpace(c.CaptureManifestURL); configured != "" {
		return configured
	}

	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		c.GetGitHubOwner(), c.GetGitHubRepo(), captureStableTag, captureManifestAsset)
}

func (c *Config) GetCheckInterval() time.Duration {
	if c.CheckInterval == 0 {
		return 1 * time.Hour
	}
	return c.CheckInterval
}

func (c *Config) GetGitHubOwner() string {
	if c.GitHubOwner == "" {
		return "emoss08"
	}
	return c.GitHubOwner
}

func (c *Config) GetGitHubRepo() string {
	if c.GitHubRepo == "" {
		return "trenova"
	}
	return c.GitHubRepo
}
