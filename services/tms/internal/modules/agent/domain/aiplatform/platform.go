package aiplatform

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// Platform is the service that serves a Claude model for an Anthropic Messages
// provider. Every platform speaks the Messages API; they differ in where it is
// reached and how a request proves who sent it.
type Platform string

const (
	// PlatformDirect calls the provider's base URL with its API key: the Claude
	// API itself, or anything that serves the Messages API the same way.
	PlatformDirect = Platform("Direct")
	// PlatformBedrock is Amazon Bedrock's Messages endpoint.
	PlatformBedrock = Platform("Bedrock")
	// PlatformVertexAI is Google Vertex AI's Anthropic publisher models.
	PlatformVertexAI = Platform("VertexAI")
	// PlatformClaudeAWS is Claude Platform on AWS, the Claude API run by
	// Anthropic and billed through AWS.
	PlatformClaudeAWS = Platform("ClaudePlatformAWS")
	// PlatformClaudeGoogleCloud is Claude Platform on Google Cloud.
	PlatformClaudeGoogleCloud = Platform("ClaudePlatformGoogleCloud")
	// PlatformFoundry is Microsoft Foundry.
	PlatformFoundry = Platform("Foundry")
)

func AllPlatforms() []Platform {
	return []Platform{
		PlatformDirect,
		PlatformBedrock,
		PlatformVertexAI,
		PlatformClaudeAWS,
		PlatformClaudeGoogleCloud,
		PlatformFoundry,
	}
}

func (p Platform) IsValid() bool {
	return slices.Contains(AllPlatforms(), p)
}

// ServesAdvisor reports whether the platform takes Anthropic's advisor tool.
// Only the Claude API and Claude Platform on AWS do; Bedrock, Vertex AI and
// Foundry refuse a request that declares it.
func (p Platform) ServesAdvisor() bool {
	return p == PlatformDirect || p == PlatformClaudeAWS
}

// PlatformAuth is how a request to a platform proves who sent it.
type PlatformAuth string

const (
	// PlatformAuthAPIKey sends the stored key: an Anthropic key, a Bedrock API
	// key, or a Foundry key.
	PlatformAuthAPIKey = PlatformAuth("ApiKey")
	// PlatformAuthAccessKey signs each request with an IAM access key; the
	// secret access key is the stored credential.
	PlatformAuthAccessKey = PlatformAuth("AccessKey")
	// PlatformAuthServiceAccount mints tokens from a Google service account
	// key, which is the stored credential.
	PlatformAuthServiceAccount = PlatformAuth("ServiceAccount")
	// PlatformAuthEntraID takes tokens from Microsoft Entra ID for an app
	// registration; its client secret is the stored credential.
	PlatformAuthEntraID = PlatformAuth("EntraID")
	// PlatformAuthServer uses the cloud credentials of the server Trenova runs
	// on. Nothing is stored, and a server allows it only when its operator
	// says so.
	PlatformAuthServer = PlatformAuth("Server")
)

func AllPlatformAuths() []PlatformAuth {
	return []PlatformAuth{
		PlatformAuthAPIKey,
		PlatformAuthAccessKey,
		PlatformAuthServiceAccount,
		PlatformAuthEntraID,
		PlatformAuthServer,
	}
}

func (a PlatformAuth) IsValid() bool {
	return slices.Contains(AllPlatformAuths(), a)
}

// Auths are the ways a platform takes, the first being its default.
func (p Platform) Auths() []PlatformAuth {
	switch p {
	case PlatformDirect:
		return []PlatformAuth{PlatformAuthAPIKey}
	case PlatformBedrock, PlatformClaudeAWS:
		return []PlatformAuth{PlatformAuthAccessKey, PlatformAuthAPIKey, PlatformAuthServer}
	case PlatformVertexAI, PlatformClaudeGoogleCloud:
		return []PlatformAuth{PlatformAuthServiceAccount, PlatformAuthServer}
	case PlatformFoundry:
		return []PlatformAuth{PlatformAuthAPIKey, PlatformAuthEntraID}
	default:
		return nil
	}
}

// DefaultAuth is the way a platform is reached when none is named.
func (p Platform) DefaultAuth() PlatformAuth {
	if auths := p.Auths(); len(auths) > 0 {
		return auths[0]
	}

	return PlatformAuthAPIKey
}

// ListsModels reports whether the platform serves the Models API. Bedrock,
// Vertex AI and Foundry name models their own way and list none through it.
func (p Platform) ListsModels() bool {
	switch p {
	case PlatformDirect, PlatformClaudeAWS, PlatformClaudeGoogleCloud:
		return true
	case PlatformBedrock, PlatformVertexAI, PlatformFoundry:
		return false
	default:
		return false
	}
}

// PlatformSettings are what a platform needs besides its credential. None of
// them is secret; each one ends up in a host name, a URL or a signature, which
// is why every one is held to the exact shape its platform gives it.
type PlatformSettings struct {
	Auth PlatformAuth `json:"auth"`
	// Region is the AWS region for Bedrock and Claude Platform on AWS, the
	// Vertex AI region, and the Google Cloud location.
	Region string `json:"region,omitempty"`
	// ProjectID is the Google Cloud project.
	ProjectID string `json:"projectId,omitempty"`
	// WorkspaceID is the Anthropic workspace for Claude Platform on AWS and on
	// Google Cloud.
	WorkspaceID string `json:"workspaceId,omitempty"`
	// AccessKeyID is the IAM access key that signs requests with AccessKey.
	AccessKeyID string `json:"accessKeyId,omitempty"`
	// Resource is the Foundry resource name.
	Resource string `json:"resource,omitempty"`
	// TenantID and ClientID are the Entra ID tenant and app registration.
	TenantID string `json:"tenantId,omitempty"`
	ClientID string `json:"clientId,omitempty"`
}

const (
	googleTokenURI           = "https://oauth2.googleapis.com/token" //nolint:gosec // a public endpoint, not a credential
	maxServiceAccountKeySize = 16 << 10
	googleCloudLocation      = "global"
	maxPlainSecretLength     = 4096
)

var (
	awsRegionPattern      = regexp.MustCompile(`^[a-z]{2}(-gov|-iso[a-z]?)?-[a-z]+-\d{1,2}$`)
	gcpRegionPattern      = regexp.MustCompile(`^(global|us|eu|[a-z]+-[a-z]+\d{1,2})$`)
	gcpProjectPattern     = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
	workspacePattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)
	accessKeyIDPattern    = regexp.MustCompile(`^[A-Z0-9]{16,128}$`)
	secretAccessKeyRegexp = regexp.MustCompile(`^[A-Za-z0-9/+=]{16,128}$`)
	azureResourcePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	guidPattern           = regexp.MustCompile(
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
	)
	tenantDomainPattern = regexp.MustCompile(
		`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`,
	)
	plainSecretPattern = regexp.MustCompile(`^[\x21-\x7e]+$`)
)

// ResolveSettings are the stored settings with the platform's defaults filled.
func (p Platform) ResolveSettings(stored *PlatformSettings) PlatformSettings {
	var settings PlatformSettings
	if stored != nil {
		settings = *stored
	}
	if settings.Auth == "" {
		settings.Auth = p.DefaultAuth()
	}
	if p == PlatformClaudeGoogleCloud && settings.Region == "" {
		settings.Region = googleCloudLocation
	}

	return settings
}

// BaseURL is the address a platform serves the Messages API at, built from
// settings already held to their shapes. Empty when a setting it needs is
// missing, which validation reports by name, and for PlatformDirect, whose
// address is the protocol's own.
func (p Platform) BaseURL(settings *PlatformSettings) string {
	switch p {
	case PlatformBedrock:
		if !awsRegionPattern.MatchString(settings.Region) {
			return ""
		}
		return "https://bedrock-mantle." + settings.Region + ".api.aws/anthropic"
	case PlatformClaudeAWS:
		if !awsRegionPattern.MatchString(settings.Region) {
			return ""
		}
		return "https://aws-external-anthropic." + settings.Region + ".api.aws"
	case PlatformVertexAI:
		return vertexBaseURL(settings.Region)
	case PlatformClaudeGoogleCloud:
		if !gcpProjectPattern.MatchString(settings.ProjectID) ||
			!gcpRegionPattern.MatchString(settings.Region) ||
			!workspacePattern.MatchString(settings.WorkspaceID) {
			return ""
		}
		return fmt.Sprintf(
			"https://claude.googleapis.com/v1alpha/projects/%s/locations/%s/workspaces/%s/invoke",
			settings.ProjectID, settings.Region, settings.WorkspaceID,
		)
	case PlatformFoundry:
		if !azureResourcePattern.MatchString(settings.Resource) {
			return ""
		}
		return "https://" + settings.Resource + ".services.ai.azure.com/anthropic"
	case PlatformDirect:
		return ""
	default:
		return ""
	}
}

func vertexBaseURL(region string) string {
	switch {
	case region == "global":
		return "https://aiplatform.googleapis.com"
	case region == "us" || region == "eu":
		return "https://aiplatform." + region + ".rep.googleapis.com"
	case gcpRegionPattern.MatchString(region):
		return "https://" + region + "-aiplatform.googleapis.com"
	default:
		return ""
	}
}

type serviceAccountKey struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	TokenURI     string `json:"token_uri"`
}

func parseServiceAccountKey(secret string) (*serviceAccountKey, error) {
	var key serviceAccountKey
	if err := sonic.UnmarshalString(secret, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

// FingerprintSubject is the part of a credential whose ends are kept to tell
// two apart. A service account key's text opens and closes the same way for
// every key, so its key id stands for it.
func FingerprintSubject(auth PlatformAuth, secret string) string {
	if auth != PlatformAuthServiceAccount {
		return secret
	}
	key, err := parseServiceAccountKey(secret)
	if err != nil {
		return secret
	}

	return key.PrivateKeyID
}

// ValidateCredential checks a credential for a platform that signs in with
// auth before it is stored. Google's loader also accepts external account and
// impersonation configurations, which can read files on the server or call any
// address they name, and a service account key names its own token endpoint,
// so only a service account key that mints its tokens at Google is taken.
func ValidateCredential(auth PlatformAuth, secret string, multiErr *errortypes.MultiError) {
	switch auth {
	case PlatformAuthServer:
		multiErr.Add(
			"apiKey",
			errortypes.ErrInvalid,
			"A provider that uses the server's credentials stores no key",
		)
	case PlatformAuthAccessKey:
		if !secretAccessKeyRegexp.MatchString(secret) {
			multiErr.Add(
				"apiKey",
				errortypes.ErrInvalid,
				"Enter the secret access key AWS gave you",
			)
		}
	case PlatformAuthServiceAccount:
		validateServiceAccountKey(secret, multiErr)
	case PlatformAuthAPIKey, PlatformAuthEntraID:
		if len(secret) > maxPlainSecretLength || !plainSecretPattern.MatchString(secret) {
			multiErr.Add(
				"apiKey",
				errortypes.ErrInvalid,
				"The key cannot contain spaces or line breaks",
			)
		}
	}
}

func validateServiceAccountKey(secret string, multiErr *errortypes.MultiError) {
	if len(secret) > maxServiceAccountKeySize {
		multiErr.Add(
			"apiKey",
			errortypes.ErrInvalid,
			"A service account key is a few kilobytes at most",
		)
		return
	}

	key, err := parseServiceAccountKey(secret)
	switch {
	case err != nil:
		multiErr.Add(
			"apiKey",
			errortypes.ErrInvalid,
			"Paste the service account key file Google Cloud gave you, in JSON",
		)
	case key.Type != "service_account":
		multiErr.Add(
			"apiKey",
			errortypes.ErrInvalid,
			"Only a service account key is accepted; other Google credential files can make the server read files or call other addresses",
		)
	case key.TokenURI != "" && key.TokenURI != googleTokenURI:
		multiErr.Add(
			"apiKey",
			errortypes.ErrInvalid,
			"The key must get its tokens from Google at {0}",
			googleTokenURI,
		)
	case strings.TrimSpace(key.ClientEmail) == "" ||
		!strings.Contains(key.PrivateKey, "PRIVATE KEY") ||
		strings.TrimSpace(key.PrivateKeyID) == "":
		multiErr.Add(
			"apiKey",
			errortypes.ErrInvalid,
			"The service account key is missing its email, key id or private key",
		)
	}
}

// Validate checks the settings of a platform other than PlatformDirect: a
// sign-in the platform takes, and each setting in the shape it gives it.
// BaseURL is the address entered, which Foundry takes in place of a resource.
func (s *PlatformSettings) Validate(
	platform Platform,
	baseURL string,
	multiErr *errortypes.MultiError,
) {
	if !s.Auth.IsValid() || !slices.Contains(platform.Auths(), s.Auth) {
		multiErr.Add(
			"platformSettings.auth",
			errortypes.ErrInvalid,
			"This platform does not sign in that way",
		)
		return
	}

	validatePlatformPlace(platform, baseURL, s, multiErr)
	validatePlatformIdentity(s, multiErr)
}

func validatePlatformPlace(
	platform Platform,
	baseURL string,
	settings *PlatformSettings,
	multiErr *errortypes.MultiError,
) {
	switch platform {
	case PlatformBedrock, PlatformClaudeAWS:
		requireShape(multiErr, "platformSettings.region", settings.Region, awsRegionPattern,
			"Enter the AWS region, such as us-east-1")
	case PlatformVertexAI:
		requireShape(multiErr, "platformSettings.region", settings.Region, gcpRegionPattern,
			"Enter the Vertex AI region, such as us-east5 or global")
		requireShape(multiErr, "platformSettings.projectId", settings.ProjectID, gcpProjectPattern,
			"Enter the Google Cloud project ID")
	case PlatformClaudeGoogleCloud:
		requireShape(multiErr, "platformSettings.region", settings.Region, gcpRegionPattern,
			"Enter the Google Cloud location, such as global")
		requireShape(multiErr, "platformSettings.projectId", settings.ProjectID, gcpProjectPattern,
			"Enter the Google Cloud project ID")
	case PlatformFoundry:
		if settings.Resource == "" && strings.TrimSpace(baseURL) == "" {
			multiErr.Add("platformSettings.resource", errortypes.ErrRequired,
				"Enter the Foundry resource name, or the endpoint's address")
		}
		if settings.Resource != "" {
			if strings.TrimSpace(baseURL) != "" {
				multiErr.Add("platformSettings.resource", errortypes.ErrInvalid,
					"Enter the resource name or the endpoint's address, not both")
			}
			requireShape(
				multiErr,
				"platformSettings.resource",
				settings.Resource,
				azureResourcePattern,
				"A Foundry resource name is lowercase letters, digits and hyphens",
			)
		}
	case PlatformDirect:
	}

	if platform == PlatformClaudeAWS || platform == PlatformClaudeGoogleCloud {
		requireShape(
			multiErr,
			"platformSettings.workspaceId",
			settings.WorkspaceID,
			workspacePattern,
			"Enter the Anthropic workspace ID",
		)
	} else if settings.WorkspaceID != "" {
		multiErr.Add("platformSettings.workspaceId", errortypes.ErrInvalid,
			"Only Claude Platform on AWS and on Google Cloud take a workspace")
	}
	if platform != PlatformVertexAI && platform != PlatformClaudeGoogleCloud &&
		settings.ProjectID != "" {
		multiErr.Add("platformSettings.projectId", errortypes.ErrInvalid,
			"Only a Google Cloud platform takes a project")
	}
	if platform != PlatformFoundry && settings.Resource != "" {
		multiErr.Add("platformSettings.resource", errortypes.ErrInvalid,
			"Only Foundry takes a resource name")
	}
	if platform == PlatformFoundry && settings.Region != "" {
		multiErr.Add("platformSettings.region", errortypes.ErrInvalid,
			"Foundry takes a resource name, not a region")
	}
}

func validatePlatformIdentity(settings *PlatformSettings, multiErr *errortypes.MultiError) {
	if settings.Auth == PlatformAuthAccessKey {
		requireShape(
			multiErr,
			"platformSettings.accessKeyId",
			settings.AccessKeyID,
			accessKeyIDPattern,
			"Enter the access key ID, the one that begins with AKIA",
		)
	} else if settings.AccessKeyID != "" {
		multiErr.Add("platformSettings.accessKeyId", errortypes.ErrInvalid,
			"Only an access key sign-in takes an access key ID")
	}

	if settings.Auth == PlatformAuthEntraID {
		if settings.TenantID == "" ||
			(!guidPattern.MatchString(settings.TenantID) &&
				!tenantDomainPattern.MatchString(settings.TenantID)) {
			multiErr.Add("platformSettings.tenantId", errortypes.ErrInvalid,
				"Enter the Entra ID tenant ID, or its primary domain")
		}
		requireShape(multiErr, "platformSettings.clientId", settings.ClientID, guidPattern,
			"Enter the application (client) ID of the app registration")
		return
	}
	if settings.TenantID != "" || settings.ClientID != "" {
		multiErr.Add("platformSettings.tenantId", errortypes.ErrInvalid,
			"Only an Entra ID sign-in takes a tenant and client ID")
	}
}

func requireShape(
	multiErr *errortypes.MultiError,
	field string,
	value string,
	pattern *regexp.Regexp,
	message string,
) {
	if value == "" {
		multiErr.Add(field, errortypes.ErrRequired, message)
		return
	}
	if !pattern.MatchString(value) {
		multiErr.Add(field, errortypes.ErrInvalid, message)
	}
}

// platformVendors name who serves a call made through a platform.
var platformVendors = map[Platform]string{
	PlatformDirect:            "",
	PlatformBedrock:           "bedrock",
	PlatformVertexAI:          "vertex",
	PlatformClaudeAWS:         "anthropic",
	PlatformClaudeGoogleCloud: "anthropic",
	PlatformFoundry:           "foundry",
}

// Vendor is who serves a call made through the platform, empty for a
// provider called directly.
func (p Platform) Vendor() string {
	return platformVendors[p]
}

// Normalize trims every setting and fills the sign-in the platform takes by
// default, so what is stored is exactly what is used.
func (s *PlatformSettings) Normalize(platform Platform) {
	s.Region = strings.ToLower(strings.TrimSpace(s.Region))
	s.ProjectID = strings.ToLower(strings.TrimSpace(s.ProjectID))
	s.WorkspaceID = strings.TrimSpace(s.WorkspaceID)
	s.AccessKeyID = strings.ToUpper(strings.TrimSpace(s.AccessKeyID))
	s.Resource = strings.ToLower(strings.TrimSpace(s.Resource))
	s.TenantID = strings.ToLower(strings.TrimSpace(s.TenantID))
	s.ClientID = strings.ToLower(strings.TrimSpace(s.ClientID))
	if s.Auth == "" && platform != PlatformDirect {
		s.Auth = platform.DefaultAuth()
	}
}
