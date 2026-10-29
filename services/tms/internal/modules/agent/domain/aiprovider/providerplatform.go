package aiprovider

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// ResolvedPlatform is the platform a provider is reached through. Only the
// Anthropic protocol has more than one.
func (p *Provider) ResolvedPlatform() aiplatform.Platform {
	if p.Kind != KindAnthropicMessages || p.Platform == "" {
		return aiplatform.PlatformDirect
	}

	return p.Platform
}

// ResolvedSettings are the platform settings with their defaults filled.
func (p *Provider) ResolvedSettings() aiplatform.PlatformSettings {
	return p.ResolvedPlatform().ResolveSettings(p.PlatformSettings)
}

// UsesServerCredentials reports a provider that signs in with the server's own
// cloud credentials.
func (p *Provider) UsesServerCredentials() bool {
	return p.ResolvedPlatform() != aiplatform.PlatformDirect &&
		p.ResolvedSettings().Auth == aiplatform.PlatformAuthServer
}

// RequiresCredential reports whether a stored credential must be present.
func (p *Provider) RequiresCredential() bool {
	if p.ResolvedPlatform() == aiplatform.PlatformDirect {
		return p.Kind.RequiresAPIKey()
	}

	return p.ResolvedSettings().Auth != aiplatform.PlatformAuthServer
}

// platformBaseURL is the address a platform serves the Messages API at, or the
// protocol's own address for a provider called directly.
func (p *Provider) platformBaseURL() string {
	platform := p.ResolvedPlatform()
	if platform == aiplatform.PlatformDirect || !platform.IsValid() {
		return p.Kind.DefaultBaseURL()
	}

	settings := p.ResolvedSettings()

	return platform.BaseURL(&settings)
}

// CredentialFingerprintSubject is the part of a credential whose ends are kept
// to tell two apart.
func (p *Provider) CredentialFingerprintSubject(secret string) string {
	if p.ResolvedPlatform() == aiplatform.PlatformDirect {
		return secret
	}

	return aiplatform.FingerprintSubject(p.ResolvedSettings().Auth, secret)
}

// ValidateCredential checks a credential before it is stored, against the way
// the provider's platform signs in.
func (p *Provider) ValidateCredential(secret string, multiErr *errortypes.MultiError) {
	secret = strings.TrimSpace(secret)
	if secret == "" || p.ResolvedPlatform() == aiplatform.PlatformDirect {
		return
	}

	aiplatform.ValidateCredential(p.ResolvedSettings().Auth, secret, multiErr)
}

func (p *Provider) validatePlatform(multiErr *errortypes.MultiError) {
	if p.Platform == "" {
		p.Platform = aiplatform.PlatformDirect
	}
	if !p.Platform.IsValid() {
		multiErr.Add("platform", errortypes.ErrInvalid, "Platform is invalid")
		return
	}
	if p.Kind != KindAnthropicMessages {
		if p.Platform != aiplatform.PlatformDirect {
			multiErr.Add(
				"platform",
				errortypes.ErrInvalid,
				"Only an Anthropic Messages provider is reached through a platform",
			)
		}
		if p.PlatformSettings != nil {
			multiErr.Add(
				"platformSettings",
				errortypes.ErrInvalid,
				"Only an Anthropic Messages provider takes platform settings",
			)
		}
		return
	}
	if p.Platform == aiplatform.PlatformDirect {
		if p.PlatformSettings != nil {
			multiErr.Add(
				"platformSettings",
				errortypes.ErrInvalid,
				"A provider called directly takes no platform settings",
			)
		}
		return
	}

	settings := p.ResolvedSettings()
	settings.Validate(p.Platform, p.BaseURL, multiErr)
}

// NormalizePlatformSettings trims every setting and fills the sign-in a
// platform takes by default, so what is stored is exactly what is used.
func (p *Provider) NormalizePlatformSettings() {
	if p.Platform == "" {
		p.Platform = aiplatform.PlatformDirect
	}
	if p.PlatformSettings != nil {
		p.PlatformSettings.Normalize(p.Platform)
	}
}

// ConnectionSnapshot copies where a provider sends its credential, taken
// before an edit so the edit can be compared against it.
func (p *Provider) ConnectionSnapshot() *Provider {
	snapshot := &Provider{Kind: p.Kind, BaseURL: p.BaseURL, Platform: p.Platform}
	if p.PlatformSettings != nil {
		settings := *p.PlatformSettings
		snapshot.PlatformSettings = &settings
	}

	return snapshot
}
