package extensionportal

import (
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/urlutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	PortalSiteIDPrefix = "apst_"
	PortalRunIDPrefix  = "aprn_"

	MaxPortalSiteNameRunes    = 100
	MaxPortalSitePurposeRunes = 500
	MaxPortalSiteDomains      = 10
	MaxPortalSiteURLLength    = 2048
	MaxPortalSitesPerTenant   = 100
	MaxPortalInstructionRunes = 2000
	MaxPortalDetails          = 20
	MaxPortalDetailKeyRunes   = 60
	MaxPortalDetailValueRunes = 500
	MaxPortalAnswerRunes      = 12000
	MaxPortalFailureRunes     = 1000
	MaxPortalVisitedURLs      = 200

	PortalCredentialUsername = "username"
	PortalCredentialPassword = "password"
	PortalSecretUsername     = "portal_username"
	PortalSecretPassword     = "portal_password"
)

func PortalCredentialFields() []configspec.Field {
	return []configspec.Field{
		{
			Key:       PortalCredentialUsername,
			Label:     "Username",
			Type:      configspec.FieldTypeString,
			Sensitive: true,
		},
		{
			Key:       PortalCredentialPassword,
			Label:     "Password",
			Type:      configspec.FieldTypePassword,
			Sensitive: true,
		},
		authenticatorField(),
	}
}

type PortalSite struct {
	keel.Table `keel:"agent_portal_sites"`

	ID               pulid.ID       `json:"id"               keel:"id,pk,autoid:apst_"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID       `json:"organizationId"   keel:"organization_id,pk,tenant"`
	Name             string         `json:"name"             keel:"name"`
	Purpose          string         `json:"purpose"          keel:"purpose"`
	Domains          []string       `json:"domains"          keel:"domains,jsonb"`
	StartURL         string         `json:"startUrl"         keel:"start_url"`
	Credentials      map[string]any `json:"-"                keel:"credentials,jsonb"`
	BrowserProfileID string         `json:"browserProfileId" keel:"browser_profile_id"`
	AllowActions     bool           `json:"allowActions"     keel:"allow_actions"`
	Enabled          bool           `json:"enabled"          keel:"enabled"`
	CreatedByID      pulid.ID       `json:"createdById"      keel:"created_by_id,nullzero"`
	Version          int64          `json:"version"          keel:"version,version"`
	CreatedAt        int64          `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64          `json:"updatedAt"        keel:"updated_at,autoupdate"`
}

func (s *PortalSite) HasSignIn() bool {
	return s != nil && configspec.HasRequired(s.Credentials, []configspec.Field{
		{Key: PortalCredentialUsername, Required: true},
		{Key: PortalCredentialPassword, Required: true},
	})
}

func (s *PortalSite) Normalize() {
	s.Name = strings.TrimSpace(s.Name)
	s.Purpose = strings.TrimSpace(s.Purpose)
	s.StartURL = strings.TrimSpace(s.StartURL)
	s.BrowserProfileID = strings.TrimSpace(s.BrowserProfileID)
	s.Domains = agentextension.NormalizeHostnames(s.Domains)
}

func (s *PortalSite) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.Name,
			validation.Required.Error("Name is required"),
			validation.RuneLength(1, MaxPortalSiteNameRunes).
				Error("Name must be 100 characters or fewer"),
		),
		validation.Field(&s.Purpose,
			validation.RuneLength(0, MaxPortalSitePurposeRunes).
				Error("What it is for must be 500 characters or fewer"),
		),
		validation.Field(&s.StartURL,
			validation.Required.Error("Start page is required"),
		),
	))

	if len(s.Domains) == 0 {
		multiErr.Add("domains", errortypes.ErrRequired,
			"Name at least one site address, such as portal.example.com")
	}
	if len(s.Domains) > MaxPortalSiteDomains {
		multiErr.Add("domains", errortypes.ErrInvalid,
			"A site can have at most 10 addresses")
	}
	for idx, domain := range s.Domains {
		if !urlutils.IsHostPattern(domain) {
			multiErr.Add(
				"domains",
				errortypes.ErrInvalid,
				"Site addresses must be hostnames such as portal.example.com or *.example.com; {0} is not",
				s.Domains[idx],
			)
		}
	}

	if s.StartURL != "" {
		if err := s.checkStartURL(); err != "" {
			multiErr.Add("startUrl", errortypes.ErrInvalid, err)
		}
	}
	if utf8.RuneCountInString(s.BrowserProfileID) > 100 {
		multiErr.Add("browserProfileId", errortypes.ErrInvalid,
			"Browser profile must be 100 characters or fewer")
	}
}

func (s *PortalSite) checkStartURL() string {
	if len(s.StartURL) > MaxPortalSiteURLLength {
		return "Start page must be 2,048 characters or fewer"
	}
	parsed, err := url.Parse(s.StartURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "Start page must be an https address, such as https://portal.example.com/login"
	}
	if !HostAllowed(parsed.Hostname(), s.Domains) {
		return "Start page must be on one of the site's addresses"
	}

	return ""
}

func HostAllowed(host string, domains []string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" {
		return false
	}
	for _, domain := range domains {
		base := strings.TrimPrefix(domain, "*.")
		if host == base || strings.HasSuffix(host, "."+base) {
			return true
		}
	}

	return false
}

func URLAllowed(raw string, domains []string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	switch parsed.Scheme {
	case "about", "data", "blob", "chrome", "chrome-error":
		return true
	case "http", "https":
		return HostAllowed(parsed.Hostname(), domains)
	default:
		return false
	}
}

type PortalRunMode string

const (
	PortalRunModeRead = PortalRunMode("Read")
	PortalRunModeAct  = PortalRunMode("Act")
)

func (m PortalRunMode) String() string { return string(m) }

func (m PortalRunMode) IsValid() bool {
	return m == PortalRunModeRead || m == PortalRunModeAct
}

func AllPortalRunModes() []PortalRunMode {
	return []PortalRunMode{PortalRunModeRead, PortalRunModeAct}
}

type PortalRunStatus string

const (
	PortalRunStatusRunning   = PortalRunStatus("Running")
	PortalRunStatusSucceeded = PortalRunStatus("Succeeded")
	PortalRunStatusFailed    = PortalRunStatus("Failed")
	PortalRunStatusStopped   = PortalRunStatus("Stopped")
	PortalRunStatusTimedOut  = PortalRunStatus("TimedOut")
	PortalRunStatusStrayed   = PortalRunStatus("Strayed")

	// PortalRunStatusAwaitingCode is a task the portal stopped for a sign-in
	// code it sent a person: the browser is kept open on the code prompt
	// until they enter it or it is too late.
	PortalRunStatusAwaitingCode = PortalRunStatus("AwaitingCode")
)

func AllPortalRunStatuses() []PortalRunStatus {
	return []PortalRunStatus{
		PortalRunStatusRunning,
		PortalRunStatusSucceeded,
		PortalRunStatusFailed,
		PortalRunStatusStopped,
		PortalRunStatusTimedOut,
		PortalRunStatusStrayed,
		PortalRunStatusAwaitingCode,
	}
}

func (s PortalRunStatus) String() string { return string(s) }

func (s PortalRunStatus) IsValid() bool {
	return slices.Contains(AllPortalRunStatuses(), s)
}

func (s PortalRunStatus) Finished() bool {
	return s.IsValid() && s != PortalRunStatusRunning && s != PortalRunStatusAwaitingCode
}

type PortalRun struct {
	keel.Table `keel:"agent_portal_runs"`

	ID                pulid.ID            `json:"id"                keel:"id,pk,autoid:aprn_"`
	BusinessUnitID    pulid.ID            `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID            `json:"organizationId"    keel:"organization_id,pk,tenant"`
	SiteID            pulid.ID            `json:"siteId"            keel:"site_id"`
	ExtensionType     agentextension.Type `json:"extensionType"     keel:"extension_type"`
	Mode              PortalRunMode       `json:"mode"              keel:"mode"`
	Status            PortalRunStatus     `json:"status"            keel:"status"`
	Instruction       string              `json:"instruction"       keel:"instruction"`
	Details           map[string]string   `json:"details"           keel:"details,jsonb"`
	UsedSignIn        bool                `json:"usedSignIn"        keel:"used_sign_in"`
	VendorTaskID      string              `json:"vendorTaskId"      keel:"vendor_task_id"`
	VendorSessionID   string              `json:"vendorSessionId"   keel:"vendor_session_id"`
	Answer            string              `json:"answer"            keel:"answer"`
	VisitedURLs       []string            `json:"visitedUrls"       keel:"visited_urls,jsonb"`
	OffSiteURLs       []string            `json:"offSiteUrls"       keel:"off_site_urls,jsonb"`
	Steps             int                 `json:"steps"             keel:"steps"`
	CostUSD           decimal.Decimal     `json:"costUsd"           keel:"cost_usd"`
	Failure           string              `json:"failure"           keel:"failure"`
	CodePrompt        string              `json:"codePrompt"        keel:"code_prompt"`
	CodeRequestedAt   *int64              `json:"codeRequestedAt"   keel:"code_requested_at"`
	CodeAttempts      int                 `json:"codeAttempts"      keel:"code_attempts"`
	IdempotencyKey    string              `json:"-"                 keel:"idempotency_key,nullzero"`
	AgentDefinitionID pulid.ID            `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	AgentRunID        pulid.ID            `json:"agentRunId"        keel:"agent_run_id,nullzero"`
	ThreadID          pulid.ID            `json:"threadId"          keel:"thread_id,nullzero"`
	ProposalID        pulid.ID            `json:"proposalId"        keel:"proposal_id,nullzero"`
	RequestedByID     pulid.ID            `json:"requestedById"     keel:"requested_by_id,nullzero"`
	StartedAt         int64               `json:"startedAt"         keel:"started_at"`
	FinishedAt        *int64              `json:"finishedAt"        keel:"finished_at"`
	Version           int64               `json:"version"           keel:"version,version"`
	CreatedAt         int64               `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64               `json:"updatedAt"         keel:"updated_at,autoupdate"`
}

func (r *PortalRun) Finish(outcome *PortalRunOutcome, at int64) {
	r.Status = outcome.Status
	r.Answer = stringutils.TruncateRunes(outcome.Answer, MaxPortalAnswerRunes)
	r.Failure = stringutils.TruncateRunes(outcome.Failure, MaxPortalFailureRunes)
	r.VisitedURLs = boundedURLs(outcome.VisitedURLs)
	r.OffSiteURLs = boundedURLs(outcome.OffSiteURLs)
	r.Steps = outcome.Steps
	r.CostUSD = outcome.CostUSD
	finished := at
	r.FinishedAt = &finished
}

type PortalRunOutcome struct {
	Status      PortalRunStatus
	Answer      string
	Failure     string
	VisitedURLs []string
	OffSiteURLs []string
	Steps       int
	CostUSD     decimal.Decimal
}

func OffSite(visited []string, domains []string) []string {
	off := make([]string, 0)
	for _, target := range visited {
		if strings.TrimSpace(target) == "" || URLAllowed(target, domains) {
			continue
		}
		if !slices.Contains(off, target) {
			off = append(off, target)
		}
	}

	return off
}

func boundedURLs(urls []string) []string {
	if urls == nil {
		return []string{}
	}
	if len(urls) > MaxPortalVisitedURLs {
		return urls[:MaxPortalVisitedURLs]
	}

	return urls
}
