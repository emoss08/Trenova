package agentextension

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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

var (
	_ bun.BeforeAppendModelHook = (*PortalSite)(nil)
	_ bun.BeforeAppendModelHook = (*PortalRun)(nil)
)

type PortalSite struct {
	bun.BaseModel `bun:"table:agent_portal_sites,alias:apst" json:"-"`

	ID               pulid.ID       `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID       `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name             string         `json:"name"             bun:"name,type:VARCHAR(100),notnull"`
	Purpose          string         `json:"purpose"          bun:"purpose,type:VARCHAR(500),notnull,default:''"`
	Domains          []string       `json:"domains"          bun:"domains,type:JSONB,notnull,default:'[]'::jsonb"`
	StartURL         string         `json:"startUrl"         bun:"start_url,type:VARCHAR(2048),notnull"`
	Credentials      map[string]any `json:"-"                bun:"credentials,type:JSONB,notnull,default:'{}'::jsonb"`
	BrowserProfileID string         `json:"browserProfileId" bun:"browser_profile_id,type:VARCHAR(100),notnull,default:''"`
	AllowActions     bool           `json:"allowActions"     bun:"allow_actions,type:BOOLEAN,notnull,default:false"`
	Enabled          bool           `json:"enabled"          bun:"enabled,type:BOOLEAN,notnull"`
	CreatedByID      pulid.ID       `json:"createdById"      bun:"created_by_id,type:VARCHAR(100),nullzero"`
	Version          int64          `json:"version"          bun:"version,type:BIGINT,notnull"`
	CreatedAt        int64          `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64          `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (s *PortalSite) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew(PortalSiteIDPrefix)
		}
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
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
	s.Domains = NormalizeHostnames(s.Domains)
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
		if !hostnamePattern.MatchString(domain) {
			multiErr.Add("domains", errortypes.ErrInvalid,
				"Site addresses must be hostnames such as portal.example.com or *.example.com; {0} is not",
				s.Domains[idx])
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

func NormalizeHostnames(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
		if host == "" || slices.Contains(out, host) {
			continue
		}
		out = append(out, host)
	}

	return out
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
)

func AllPortalRunStatuses() []PortalRunStatus {
	return []PortalRunStatus{
		PortalRunStatusRunning,
		PortalRunStatusSucceeded,
		PortalRunStatusFailed,
		PortalRunStatusStopped,
		PortalRunStatusTimedOut,
		PortalRunStatusStrayed,
	}
}

func (s PortalRunStatus) String() string { return string(s) }

func (s PortalRunStatus) IsValid() bool {
	return slices.Contains(AllPortalRunStatuses(), s)
}

func (s PortalRunStatus) Finished() bool {
	return s.IsValid() && s != PortalRunStatusRunning
}

type PortalRun struct {
	bun.BaseModel `bun:"table:agent_portal_runs,alias:aprn" json:"-"`

	ID                pulid.ID          `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID          `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	SiteID            pulid.ID          `json:"siteId"            bun:"site_id,type:VARCHAR(100),notnull"`
	ExtensionType     Type              `json:"extensionType"     bun:"extension_type,type:VARCHAR(50),notnull"`
	Mode              PortalRunMode     `json:"mode"              bun:"mode,type:VARCHAR(10),notnull"`
	Status            PortalRunStatus   `json:"status"            bun:"status,type:VARCHAR(20),notnull"`
	Instruction       string            `json:"instruction"       bun:"instruction,type:TEXT,notnull"`
	Details           map[string]string `json:"details"           bun:"details,type:JSONB,notnull,default:'{}'::jsonb"`
	UsedSignIn        bool              `json:"usedSignIn"        bun:"used_sign_in,type:BOOLEAN,notnull,default:false"`
	VendorTaskID      string            `json:"vendorTaskId"      bun:"vendor_task_id,type:VARCHAR(100),notnull,default:''"`
	VendorSessionID   string            `json:"vendorSessionId"   bun:"vendor_session_id,type:VARCHAR(100),notnull,default:''"`
	Answer            string            `json:"answer"            bun:"answer,type:TEXT,notnull,default:''"`
	VisitedURLs       []string          `json:"visitedUrls"       bun:"visited_urls,type:JSONB,notnull,default:'[]'::jsonb"`
	OffSiteURLs       []string          `json:"offSiteUrls"       bun:"off_site_urls,type:JSONB,notnull,default:'[]'::jsonb"`
	Steps             int               `json:"steps"             bun:"steps,type:INTEGER,notnull,default:0"`
	CostUSD           decimal.Decimal   `json:"costUsd"           bun:"cost_usd,type:NUMERIC(19,6),notnull,default:0"`
	Failure           string            `json:"failure"           bun:"failure,type:TEXT,notnull,default:''"`
	IdempotencyKey    string            `json:"-"                 bun:"idempotency_key,type:VARCHAR(200),nullzero"`
	AgentDefinitionID pulid.ID          `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	AgentRunID        pulid.ID          `json:"agentRunId"        bun:"agent_run_id,type:VARCHAR(100),nullzero"`
	ThreadID          pulid.ID          `json:"threadId"          bun:"thread_id,type:VARCHAR(100),nullzero"`
	ProposalID        pulid.ID          `json:"proposalId"        bun:"proposal_id,type:VARCHAR(100),nullzero"`
	RequestedByID     pulid.ID          `json:"requestedById"     bun:"requested_by_id,type:VARCHAR(100),nullzero"`
	StartedAt         int64             `json:"startedAt"         bun:"started_at,type:BIGINT,notnull"`
	FinishedAt        *int64            `json:"finishedAt"        bun:"finished_at,type:BIGINT,nullzero"`
	Version           int64             `json:"version"           bun:"version,type:BIGINT,notnull"`
	CreatedAt         int64             `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64             `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *PortalRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew(PortalRunIDPrefix)
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
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
