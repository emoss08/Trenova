package extensionwebwatch

import (
	"strings"

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
	WebWatchIDPrefix        = "awwt_"
	WebWatchFindingIDPrefix = "awwf_"

	DefaultWebWatchDays     = 30
	MinWebWatchDays         = 1
	MaxWebWatchDays         = 90
	MaxWebWatchQueryRunes   = agentextension.MaxSearchQueryLength
	MaxWebWatchPurposeRunes = 300
	MaxWebWatchDomains      = 20
	MaxWebWatchFailureRunes = 500
	MaxWebWatchSummaryRunes = 4000
	MaxWebWatchExcerptRunes = 600
	MaxWebWatchTitleRunes   = 300
	MaxWebWatchURLLength    = 2048
	WebWatchResultsPerRun   = 10

	// WebWatchSecretField holds the vendor's webhook secret, encrypted like
	// any other extension secret.
	WebWatchSecretField = "webhookSecret"
)

var (

	// ExaMonitorRunCostUSD is Exa's published price for one monitor run of up
	// to ten results ($15 per 1,000); Exa does not report what a run cost.
	ExaMonitorRunCostUSD = decimal.RequireFromString("0.015")
)

type WebWatchCadence string

const (
	WebWatchCadenceSixHours = WebWatchCadence("SixHours")
	WebWatchCadenceDaily    = WebWatchCadence("Daily")
	WebWatchCadenceWeekly   = WebWatchCadence("Weekly")
)

func AllWebWatchCadences() []WebWatchCadence {
	return []WebWatchCadence{WebWatchCadenceSixHours, WebWatchCadenceDaily, WebWatchCadenceWeekly}
}

func (c WebWatchCadence) String() string { return string(c) }

func (c WebWatchCadence) IsValid() bool {
	switch c {
	case WebWatchCadenceSixHours, WebWatchCadenceDaily, WebWatchCadenceWeekly:
		return true
	default:
		return false
	}
}

// Seconds is how often a watch with this cadence looks.
func (c WebWatchCadence) Seconds() int64 {
	switch c {
	case WebWatchCadenceSixHours:
		return 6 * 60 * 60
	case WebWatchCadenceDaily:
		return 24 * 60 * 60
	case WebWatchCadenceWeekly:
		return 7 * 24 * 60 * 60
	default:
		return 0
	}
}

// Words is how the cadence reads in a sentence: "every six hours".
func (c WebWatchCadence) Words() string {
	switch c {
	case WebWatchCadenceSixHours:
		return "every six hours"
	case WebWatchCadenceDaily:
		return "once a day"
	case WebWatchCadenceWeekly:
		return "once a week"
	default:
		return ""
	}
}

type WebWatchStatus string

const (
	WebWatchStatusActive = WebWatchStatus("Active")
	// WebWatchStatusEnded is a watch whose time ran out.
	WebWatchStatusEnded = WebWatchStatus("Ended")
	// WebWatchStatusStopped is a watch a person or an agent stopped.
	WebWatchStatusStopped = WebWatchStatus("Stopped")
	// WebWatchStatusFailed is a watch the vendor could no longer run.
	WebWatchStatusFailed = WebWatchStatus("Failed")
)

func AllWebWatchStatuses() []WebWatchStatus {
	return []WebWatchStatus{
		WebWatchStatusActive,
		WebWatchStatusEnded,
		WebWatchStatusStopped,
		WebWatchStatusFailed,
	}
}

func (s WebWatchStatus) String() string { return string(s) }

func (s WebWatchStatus) IsValid() bool {
	switch s {
	case WebWatchStatusActive, WebWatchStatusEnded, WebWatchStatusStopped, WebWatchStatusFailed:
		return true
	default:
		return false
	}
}

// WebWatch is a search the vendor runs on a schedule for an agent, which
// hands back only what earlier runs did not find. It belongs to the work
// that set it: the conversation it answers, or a background run's agent.
type WebWatch struct {
	keel.Table `keel:"agent_web_watches"`

	ID                pulid.ID            `json:"id"                keel:"id,pk,autoid:awwt_"`
	BusinessUnitID    pulid.ID            `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID            `json:"organizationId"    keel:"organization_id,pk,tenant"`
	ExtensionType     agentextension.Type `json:"extensionType"     keel:"extension_type"`
	VendorMonitorID   string              `json:"-"                 keel:"vendor_monitor_id"`
	Query             string              `json:"query"             keel:"query"`
	Purpose           string              `json:"purpose"           keel:"purpose"`
	IncludeDomains    []string            `json:"includeDomains"    keel:"include_domains,jsonb"`
	Cadence           WebWatchCadence     `json:"cadence"           keel:"cadence"`
	Status            WebWatchStatus      `json:"status"            keel:"status"`
	Secrets           map[string]any      `json:"-"                 keel:"secrets,jsonb"`
	EndsAt            int64               `json:"endsAt"            keel:"ends_at"`
	LastCheckedAt     *int64              `json:"lastCheckedAt"     keel:"last_checked_at"`
	LastRunAt         *int64              `json:"lastRunAt"         keel:"last_run_at"`
	RunCount          int                 `json:"runCount"          keel:"run_count"`
	FindingCount      int                 `json:"findingCount"      keel:"finding_count"`
	CostUSD           decimal.Decimal     `json:"costUsd"           keel:"cost_usd"`
	Failure           string              `json:"failure"           keel:"failure"`
	AgentDefinitionID pulid.ID            `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	AgentRunID        pulid.ID            `json:"agentRunId"        keel:"agent_run_id,nullzero"`
	ThreadID          pulid.ID            `json:"threadId"          keel:"thread_id,nullzero"`
	RequestedByID     pulid.ID            `json:"requestedById"     keel:"requested_by_id,nullzero"`
	StoppedByID       pulid.ID            `json:"stoppedById"       keel:"stopped_by_id,nullzero"`
	FinishedAt        *int64              `json:"finishedAt"        keel:"finished_at"`
	Version           int64               `json:"version"           keel:"version,version"`
	CreatedAt         int64               `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64               `json:"updatedAt"         keel:"updated_at,autoupdate"`
}

func (w *WebWatch) Normalize() {
	w.Query = strings.TrimSpace(w.Query)
	w.Purpose = strings.TrimSpace(w.Purpose)
	w.IncludeDomains = agentextension.NormalizeHostnames(w.IncludeDomains)
}

func (w *WebWatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(w,
		validation.Field(&w.Query,
			validation.Required.Error("Say what to search for"),
			validation.RuneLength(1, MaxWebWatchQueryRunes).
				Error("What to search for must be 400 characters or fewer"),
		),
		validation.Field(&w.Purpose,
			validation.Required.Error("Say what the person wants to hear about"),
			validation.RuneLength(1, MaxWebWatchPurposeRunes).
				Error("What to watch for must be 300 characters or fewer"),
		),
	))

	if !w.Cadence.IsValid() {
		multiErr.Add("cadence", errortypes.ErrInvalid,
			"How often must be every six hours, daily or weekly")
	}
	if len(w.IncludeDomains) > MaxWebWatchDomains {
		multiErr.Add("sites", errortypes.ErrInvalid, "A watch can name at most 20 sites")
	}
	for _, domain := range w.IncludeDomains {
		if strings.HasPrefix(domain, "*.") || !urlutils.IsHostPattern(domain) {
			multiErr.Add("sites", errortypes.ErrInvalid,
				"Sites must be hostnames such as fmcsa.dot.gov; {0} is not", domain)
		}
	}
}

// WatchDays reads how long a watch should last, defaulting to a month and
// bounded to a quarter so a forgotten watch does not bill indefinitely.
func WatchDays(raw int) (int, error) {
	if raw == 0 {
		return DefaultWebWatchDays, nil
	}
	if raw < MinWebWatchDays || raw > MaxWebWatchDays {
		return 0, errortypes.NewValidationError("days", errortypes.ErrInvalid,
			"A watch lasts from 1 to 90 days")
	}

	return raw, nil
}

func (w *WebWatch) Active() bool {
	return w != nil && w.Status == WebWatchStatusActive
}

// Due reports a watch whose time has run out.
func (w *WebWatch) Due(now int64) bool {
	return w.Active() && now >= w.EndsAt
}

// End records how a watch ended. A watch that already ended keeps how it
// ended first.
func (w *WebWatch) End(status WebWatchStatus, failure string, at int64) {
	if !w.Active() {
		return
	}
	w.Status = status
	w.Failure = stringutils.TruncateRunes(strings.TrimSpace(failure), MaxWebWatchFailureRunes)
	finished := at
	w.FinishedAt = &finished
}

// CountRun adds one run the vendor made, which it bills whether or not it
// found anything.
func (w *WebWatch) CountRun(createdAt int64, cost decimal.Decimal) {
	w.RunCount++
	w.CostUSD = w.CostUSD.Add(cost)
	if w.LastRunAt == nil || createdAt > *w.LastRunAt {
		ran := createdAt
		w.LastRunAt = &ran
	}
}

// NoteFailure keeps the last reason a run failed, for the person who reads
// the watch in AI Control.
func (w *WebWatch) NoteFailure(reason string) {
	w.Failure = stringutils.TruncateRunes(strings.TrimSpace(reason), MaxWebWatchFailureRunes)
}

func (w *WebWatch) HasSecret() bool {
	return w != nil && configspec.HasRequired(w.Secrets, []configspec.Field{
		{Key: WebWatchSecretField, Required: true},
	})
}

func WebWatchSecretFields() []configspec.Field {
	return []configspec.Field{{
		Key:       WebWatchSecretField,
		Label:     "Webhook secret",
		Type:      configspec.FieldTypePassword,
		Sensitive: true,
	}}
}

// WebWatchResult is one page a watch found new, as the vendor described it:
// text written outside the organization.
type WebWatchResult struct {
	Title         string `json:"title"`
	URL           string `json:"url"`
	PublishedDate string `json:"publishedDate,omitempty"`
	Excerpt       string `json:"excerpt,omitempty"`
}

// WebWatchFinding is one run that found something new: its summary and the
// pages it found, kept with the watch and handed to the work waiting on it.
type WebWatchFinding struct {
	keel.Table `keel:"agent_web_watch_findings"`

	ID             pulid.ID         `json:"id"             keel:"id,pk,autoid:awwf_"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	WatchID        pulid.ID         `json:"watchId"        keel:"watch_id"`
	VendorRunID    string           `json:"-"              keel:"vendor_run_id"`
	Summary        string           `json:"summary"        keel:"summary"`
	Results        []WebWatchResult `json:"results"        keel:"results,jsonb"`
	FoundAt        int64            `json:"foundAt"        keel:"found_at"`
	DeliveredAt    *int64           `json:"deliveredAt"    keel:"delivered_at"`
	CreatedAt      int64            `json:"createdAt"      keel:"created_at,autocreate"`
}

// BoundedResults keeps a run's results to what is stored and handed on: at
// most a run's worth, each trimmed, without pages whose address is not an
// absolute http or https one, since each is shown to a person as a link.
func BoundedResults(raw []WebWatchResult) []WebWatchResult {
	out := make([]WebWatchResult, 0, min(len(raw), WebWatchResultsPerRun))
	for idx := range raw {
		if len(out) == WebWatchResultsPerRun {
			break
		}
		target := strings.TrimSpace(raw[idx].URL)
		if len(target) > MaxWebWatchURLLength {
			continue
		}
		if _, ok := urlutils.ParseAbsoluteHTTP(target); !ok {
			continue
		}
		out = append(out, WebWatchResult{
			Title: stringutils.TruncateRunes(strings.TrimSpace(raw[idx].Title),
				MaxWebWatchTitleRunes),
			URL:           target,
			PublishedDate: strings.TrimSpace(raw[idx].PublishedDate),
			Excerpt: stringutils.TruncateRunes(strings.TrimSpace(raw[idx].Excerpt),
				MaxWebWatchExcerptRunes),
		})
	}

	return out
}

// BoundedSummary keeps a run's summary to what is stored and handed on.
func BoundedSummary(raw string) string {
	return stringutils.TruncateRunes(strings.TrimSpace(raw), MaxWebWatchSummaryRunes)
}

// WebWatchWebhookPrefix is where vendors deliver a watch's news, under the
// API's base path. The tenant is in the path so the watch is read under its
// own tenant before the delivery's signature is checked against it.
const WebWatchWebhookPrefix = "/webhooks/agent-extensions/web-watches"

func WebWatchWebhookPath(orgID, buID, watchID pulid.ID) string {
	return WebWatchWebhookPrefix + "/" + orgID.String() + "/" + buID.String() + "/" +
		watchID.String() + "/"
}

// MaxActiveWebWatches is how many watches a vendor's settings allow at once.
func MaxActiveWebWatches(typ agentextension.Type, values map[string]string) int {
	if typ != agentextension.TypeExa {
		return agentextension.DefaultMaxActiveWebWatches
	}
	settings, err := agentextension.ParseExaSettings(values)
	if err != nil {
		return agentextension.DefaultMaxActiveWebWatches
	}

	return settings.MaxActiveWatches
}

// WebWatchChecks is how many times a watch looks over its life.
func WebWatchChecks(cadence WebWatchCadence, days int) int64 {
	every := cadence.Seconds()
	if every <= 0 {
		return 0
	}

	return int64(days) * 24 * 60 * 60 / every
}
