package agentextensionservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	refLength            = 16
	refKeyLabel          = "trenova:web-result-ref:v1:"
	searchExcerptBudget  = 9000
	minExcerptCharacters = 400
	maxExcerptCharacters = 1500
	PagePartCharacters   = 8000
	MaxPageParts         = 6
	MaxRecencyDays       = 3650
)

const unknownRefMessage = "Only pages a web_search returned can be read, with the url and ref " +
	"exactly as that search gave them. Search first, then read one of its results."

const unreadablePageMessage = "The page could not be retrieved. It may have moved, require a " +
	"login, or block automated readers. Try another result."

const invalidSettingsSuffix = " extension's settings are invalid. An administrator can " +
	"correct them under AI Control, Extensions."

type settledCall struct {
	day    int
	failed bool
	cost   decimal.Decimal
}

func (s *Service) SearchWeb(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query agentextensionports.WebSearchQuery,
) (*agentextensionports.WebSearchOutcome, error) {
	text := strings.TrimSpace(query.Query)
	if err := agentextension.CheckSearchQuery(text); err != nil {
		return nil, &ToolError{Message: stringutils.CapitalizeFirst(err.Error())}
	}

	research, err := s.webResearch(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	settings := research.settings

	numResults := settings.ResultsPerSearch
	if query.ResultLimit > 0 {
		numResults = min(query.ResultLimit, settings.ResultsPerSearch)
	}

	request := &agentextensionports.WebSearchProviderRequest{
		Settings:       research.values,
		Query:          text,
		NumResults:     numResults,
		ExcludeDomains: settings.ExcludedDomains,
		ExcerptCharacters: intutils.Clamp(
			searchExcerptBudget/numResults,
			minExcerptCharacters,
			maxExcerptCharacters,
		),
	}
	now := s.now()
	if query.PublishedWithinDays > 0 {
		days := min(query.PublishedWithinDays, MaxRecencyDays)
		request.StartPublishedDate = time.Unix(timeutils.DayStartUTC(now), 0).UTC().
			AddDate(0, 0, -days).Format(time.RFC3339)
	}

	call := research.metered(tenantInfo)
	day, err := s.reserve(ctx, call)
	if err != nil {
		return nil, err
	}

	resp, err := research.provider.Search(ctx, request)
	settled := settledCall{day: day, failed: err != nil}
	if resp != nil {
		settled.cost = resp.CostUSD
	}
	s.settle(ctx, call, settled)
	if err != nil {
		return nil, s.providerError(research.failing(err, "search the web"))
	}

	key := refKey(settings.APIKey)
	hits := make([]agentextensionports.WebSearchHit, 0, len(resp.Results))
	official := make([]agentextensionports.WebSearchHit, 0, len(resp.Results))
	for idx := range resp.Results {
		result := &resp.Results[idx]
		if _, parseErr := agentextension.ParsePageURL(result.URL); parseErr != nil {
			continue
		}
		site := agentextension.SiteOf(result.URL)
		hit := agentextensionports.WebSearchHit{
			Ref:           resultRef(key, tenantInfo, result.URL),
			Title:         result.Title,
			URL:           result.URL,
			Site:          site,
			Official:      agentextension.IsOfficialSite(site),
			PublishedDate: publishedDay(result.PublishedDate),
			Author:        result.Author,
			Excerpts:      result.Excerpts,
		}
		if hit.Official {
			official = append(official, hit)
			continue
		}
		hits = append(hits, hit)
	}

	return &agentextensionports.WebSearchOutcome{
		Hits:        append(official, hits...),
		RetrievedAt: now,
	}, nil
}

func (s *Service) ReadWebPage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query agentextensionports.WebPageQuery,
) (*agentextensionports.WebPage, error) {
	target := strings.TrimSpace(query.URL)
	if _, err := agentextension.ParsePageURL(target); err != nil {
		return nil, &ToolError{Message: stringutils.CapitalizeFirst(err.Error()) + "."}
	}

	part := query.Part
	if part <= 0 {
		part = 1
	}
	if part > MaxPageParts {
		return nil, &ToolError{Message: "Only the first " + strconv.Itoa(MaxPageParts) +
			" parts of a page can be read. Search for the specific section instead."}
	}

	research, err := s.webResearch(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	if !validRef(refKey(research.settings.APIKey), tenantInfo, target, query.Ref) {
		return nil, &ToolError{Message: unknownRefMessage}
	}

	call := research.metered(tenantInfo)
	day, err := s.reserve(ctx, call)
	if err != nil {
		return nil, err
	}

	resp, err := research.provider.Read(ctx, agentextensionports.WebPageProviderRequest{
		Settings:      research.values,
		URL:           target,
		MaxCharacters: part*PagePartCharacters + 1,
	})
	settled := settledCall{day: day, failed: err != nil}
	if resp != nil {
		settled.cost = resp.CostUSD
	}
	s.settle(ctx, call, settled)
	if err != nil {
		failure := research.failing(err, "read the page")
		if failure.failure.Kind == agentextensionports.ProviderFailureNotFound {
			return nil, &ToolError{Message: unreadablePageMessage}
		}
		return nil, s.providerError(failure)
	}

	text, hasMore, inRange := pagePart(resp.Text, part)
	if !inRange {
		return nil, &ToolError{Message: "The page has no part " + strconv.Itoa(part) +
			"; it ended in an earlier part."}
	}

	address := resp.URL
	if address == "" {
		address = target
	}
	site := agentextension.SiteOf(address)

	return &agentextensionports.WebPage{
		Title:         resp.Title,
		URL:           address,
		Site:          site,
		Official:      agentextension.IsOfficialSite(site),
		PublishedDate: publishedDay(resp.PublishedDate),
		Author:        resp.Author,
		Text:          text,
		Part:          part,
		HasMore:       hasMore,
		RetrievedAt:   s.now(),
	}, nil
}

func (s *Service) TestConnection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
) (*agentextensionports.AgentExtensionTestResponse, error) {
	spec, def, err := lookup(typ)
	if err != nil {
		return nil, err
	}
	if !spec.SupportsTestConnect {
		return nil, errortypes.NewBusinessError("{0} cannot be tested", def.Name)
	}
	if err = s.requirePlan(ctx, tenantInfo, spec.Capability); err != nil {
		return nil, err
	}

	extension, err := s.settingsFor(ctx, tenantInfo, typ)
	if err != nil {
		return nil, asBusinessError(err)
	}
	multiErr := errortypes.NewMultiError()
	agentextension.ValidateSettings(typ, extension.values, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	call := meteredCall{tenant: tenantInfo, typ: typ, limit: unlimitedRequests}
	day, err := s.reserve(ctx, call)
	if err != nil {
		return nil, asBusinessError(err)
	}

	started := time.Now()
	result, err := extension.connector.Test(ctx, extension.values)
	latency := time.Since(started).Milliseconds()
	settled := settledCall{day: day, failed: err != nil}
	if result != nil {
		settled.cost = result.CostUSD
	}
	s.settle(ctx, call, settled)
	if err != nil {
		return nil, asBusinessError(s.providerError(extension.failing(err, "run a test")))
	}

	message := def.Vendor + " answered in " + strconv.FormatInt(latency, 10) + " ms."
	if result != nil && result.Note != "" {
		message += " " + result.Note
	}

	return &agentextensionports.AgentExtensionTestResponse{
		Type:      typ,
		Success:   true,
		CheckedAt: s.now(),
		LatencyMs: latency,
		Message:   message,
	}, nil
}

type webResearchCall struct {
	*extensionSettings
	settings agentextension.WebResearchSettings
	provider agentextensionports.WebSearchProvider
}

func (s *Service) webResearch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*webResearchCall, error) {
	extension, err := s.serving(ctx, tenantInfo, agentextension.CapabilityWebResearch)
	if err != nil {
		return nil, err
	}

	provider, ok := s.webSearch[extension.typ]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"the {0} extension cannot search the web", extension.typ.String(),
		)
	}

	multiErr := errortypes.NewMultiError()
	agentextension.ValidateSettings(extension.typ, extension.values, multiErr)
	settings, err := agentextension.ParseWebResearchSettings(extension.values)
	if multiErr.HasErrors() || err != nil {
		s.l.Error("stored web research settings are invalid",
			zap.String("type", extension.typ.String()), zap.Error(errors.Join(multiErr, err)))
		return nil, &ToolError{Message: "The " + extension.def.title() + invalidSettingsSuffix}
	}

	return &webResearchCall{
		extensionSettings: extension,
		settings:          settings,
		provider:          provider,
	}, nil
}

func (c *webResearchCall) metered(tenantInfo pagination.TenantInfo) meteredCall {
	return meteredCall{
		tenant: tenantInfo,
		typ:    c.typ,
		limit:  c.settings.DailyRequestLimit,
	}
}

type providerFailure struct {
	err     error
	vendor  string
	action  string
	failure agentextensionports.ProviderFailure
	spec    agentextension.CapabilitySpec
}

func (e *extensionSettings) failing(err error, action string) *providerFailure {
	spec, _ := agentextension.SpecFor(e.typ)
	capability, _ := agentextension.CapabilitySpecFor(spec.Capability)

	return &providerFailure{
		err:     err,
		vendor:  e.def.Vendor,
		action:  action,
		failure: e.connector.Classify(err),
		spec:    capability,
	}
}

func (s *Service) providerError(f *providerFailure) error {
	var message string
	switch {
	case f.failure.Kind == agentextensionports.ProviderFailureUnauthorized:
		message = f.vendor + " rejected the organization's API key, so it could not " + f.action +
			". An administrator can update the key under AI Control, Extensions."
	case f.failure.Kind == agentextensionports.ProviderFailureOutOfCredits:
		message = "The organization's " + f.vendor + " account is out of credits, so it could not " +
			f.action + ". An administrator can add credits in the " + f.vendor + " dashboard."
	case f.failure.Kind == agentextensionports.ProviderFailureRateLimited:
		message = f.vendor + " is limiting how fast requests can be made. Wait a minute before " +
			"trying again."
	case f.failure.Kind == agentextensionports.ProviderFailureInvalidRequest:
		message = f.vendor + " could not " + f.action + " as asked"
		if f.failure.Detail != "" {
			message += ": " + f.failure.Detail
		}
		message += ". Rephrase the request."
	case errors.Is(f.err, context.DeadlineExceeded), errors.Is(f.err, context.Canceled):
		message = f.vendor + " took too long to " + f.action + ". Try again, or answer without it."
	default:
		s.l.Warn("extension provider failed", zap.Error(f.err),
			zap.String("vendor", f.vendor), zap.String("action", f.action))
		message = f.vendor + " could not be reached to " + f.action + ". Try again shortly, or " +
			"answer without it and say that " + f.spec.Unchecked + "."
	}

	return &ToolError{Message: message}
}

func asBusinessError(err error) error {
	var toolErr *ToolError
	if errors.As(err, &toolErr) {
		return errortypes.NewBusinessError("{0}", toolErr.Message)
	}

	return err
}

func dailyLimitMessage(spec agentextension.CapabilitySpec, limit int) string {
	return "This organization has used its daily limit of " + strconv.Itoa(limit) + " " +
		spec.RequestNoun + ". It resets at midnight UTC. Answer from what you already have and " +
		"say that " + spec.Unchecked + " again today."
}

func refKey(apiKey string) []byte {
	sum := sha256.Sum256([]byte(refKeyLabel + apiKey))
	return sum[:]
}

func resultRef(key []byte, tenantInfo pagination.TenantInfo, target string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(tenantInfo.OrgID.String()))
	mac.Write([]byte{0})
	mac.Write([]byte(tenantInfo.BuID.String()))
	mac.Write([]byte{0})
	mac.Write([]byte(strings.TrimSpace(target)))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))[:refLength]
}

func validRef(key []byte, tenantInfo pagination.TenantInfo, target, ref string) bool {
	ref = strings.TrimSpace(ref)
	if len(ref) != refLength {
		return false
	}

	return hmac.Equal([]byte(resultRef(key, tenantInfo, target)), []byte(ref))
}

func pagePart(text string, part int) (string, bool, bool) {
	runes := []rune(text)
	start := (part - 1) * PagePartCharacters
	if start >= len(runes) && part > 1 {
		return "", false, false
	}
	end := min(start+PagePartCharacters, len(runes))

	return strings.TrimSpace(string(runes[start:end])), len(runes) > end, true
}

func publishedDay(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC().Format(time.DateOnly)
	}
	if len(value) >= len(time.DateOnly) {
		if parsed, err := time.Parse(time.DateOnly, value[:len(time.DateOnly)]); err == nil {
			return parsed.Format(time.DateOnly)
		}
	}

	return ""
}
