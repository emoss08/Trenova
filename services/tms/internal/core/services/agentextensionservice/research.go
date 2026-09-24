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

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/exa"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/restx"
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
	testQuery            = "Federal Motor Carrier Safety Administration hours of service"
	providerExa          = "Exa"
)

const unknownRefMessage = "Only pages a web_search returned can be read, with the url and ref " +
	"exactly as that search gave them. Search first, then read one of its results."

const unreadablePageMessage = "The page could not be retrieved. It may have moved, require a " +
	"login, or block automated readers. Try another result."

const invalidSettingsMessage = "The web search extension's settings are invalid. An " +
	"administrator can correct them under AI Control, Extensions."

type settledCall struct {
	day    int
	failed bool
	cost   decimal.Decimal
}

func (s *Service) SearchWeb(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query serviceports.WebSearchQuery,
) (*serviceports.WebSearchOutcome, error) {
	text := strings.TrimSpace(query.Query)
	if err := agentextension.CheckSearchQuery(text); err != nil {
		return nil, &ToolError{Message: stringutils.CapitalizeFirst(err.Error())}
	}

	settings, err := s.exaSettings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	numResults := settings.ResultsPerSearch
	if query.ResultLimit > 0 {
		numResults = min(query.ResultLimit, settings.ResultsPerSearch)
	}

	request := serviceports.WebSearchProviderRequest{
		APIKey:            settings.APIKey,
		Query:             text,
		SearchType:        settings.SearchType,
		NumResults:        numResults,
		ExcludeDomains:    settings.ExcludedDomains,
		ExcerptCharacters: intutils.Clamp(searchExcerptBudget/numResults, minExcerptCharacters, maxExcerptCharacters),
	}
	now := s.now()
	if query.PublishedWithinDays > 0 {
		days := min(query.PublishedWithinDays, MaxRecencyDays)
		request.StartPublishedDate = time.Unix(timeutils.DayStartUTC(now), 0).UTC().
			AddDate(0, 0, -days).Format(time.RFC3339)
	}

	call := meteredCall{tenant: tenantInfo, typ: agentextension.TypeExa, limit: settings.DailyRequestLimit}
	day, err := s.reserve(ctx, call)
	if err != nil {
		return nil, err
	}

	resp, err := s.webSearch.Search(ctx, request)
	settled := settledCall{day: day, failed: err != nil}
	if resp != nil {
		settled.cost = resp.CostUSD
	}
	s.settle(ctx, call, settled)
	if err != nil {
		return nil, s.providerError(err, "search the web")
	}

	key := refKey(settings.APIKey)
	hits := make([]serviceports.WebSearchHit, 0, len(resp.Results))
	official := make([]serviceports.WebSearchHit, 0, len(resp.Results))
	for idx := range resp.Results {
		result := &resp.Results[idx]
		if _, parseErr := agentextension.ParsePageURL(result.URL); parseErr != nil {
			continue
		}
		site := agentextension.SiteOf(result.URL)
		hit := serviceports.WebSearchHit{
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

	return &serviceports.WebSearchOutcome{
		Hits:        append(official, hits...),
		RetrievedAt: now,
	}, nil
}

func (s *Service) ReadWebPage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query serviceports.WebPageQuery,
) (*serviceports.WebPage, error) {
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

	settings, err := s.exaSettings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	if !validRef(refKey(settings.APIKey), tenantInfo, target, query.Ref) {
		return nil, &ToolError{Message: unknownRefMessage}
	}

	call := meteredCall{tenant: tenantInfo, typ: agentextension.TypeExa, limit: settings.DailyRequestLimit}
	day, err := s.reserve(ctx, call)
	if err != nil {
		return nil, err
	}

	resp, err := s.webSearch.Read(ctx, serviceports.WebPageProviderRequest{
		APIKey:        settings.APIKey,
		URL:           target,
		MaxCharacters: part*PagePartCharacters + 1,
	})
	settled := settledCall{day: day, failed: err != nil}
	if resp != nil {
		settled.cost = resp.CostUSD
	}
	s.settle(ctx, call, settled)
	if err != nil {
		if errors.Is(err, exa.ErrContentNotFound) {
			return nil, &ToolError{Message: unreadablePageMessage}
		}
		return nil, s.providerError(err, "read the page")
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

	return &serviceports.WebPage{
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
) (*serviceports.AgentExtensionTestResponse, error) {
	spec, def, err := lookup(typ)
	if err != nil {
		return nil, err
	}
	if !spec.SupportsTestConnect {
		return nil, errortypes.NewBusinessError("{0} cannot be tested", def.Name)
	}

	runtime, err := s.runtime(ctx, tenantInfo, typ, false)
	if err != nil {
		return nil, asBusinessError(err)
	}
	settings, err := agentextension.ParseExaSettings(runtime.values)
	if err != nil {
		return nil, err
	}

	call := meteredCall{tenant: tenantInfo, typ: typ, limit: unlimitedRequests}
	day, err := s.reserve(ctx, call)
	if err != nil {
		return nil, asBusinessError(err)
	}

	started := time.Now()
	resp, err := s.webSearch.Search(ctx, serviceports.WebSearchProviderRequest{
		APIKey:     settings.APIKey,
		Query:      testQuery,
		SearchType: agentextension.ExaSearchTypeFast,
		NumResults: 1,
	})
	latency := time.Since(started).Milliseconds()
	settled := settledCall{day: day, failed: err != nil}
	if resp != nil {
		settled.cost = resp.CostUSD
	}
	s.settle(ctx, call, settled)
	if err != nil {
		return nil, asBusinessError(s.providerError(err, "run a test search"))
	}

	message := def.Vendor + " answered in " + strconv.FormatInt(latency, 10) + " ms."
	if len(resp.Results) == 0 {
		message += " The test search returned no pages, which can happen; the key works."
	}

	return &serviceports.AgentExtensionTestResponse{
		Type:      typ,
		Success:   true,
		CheckedAt: s.now(),
		LatencyMs: latency,
		Message:   message,
	}, nil
}

func (s *Service) exaSettings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (agentextension.ExaSettings, error) {
	runtime, err := s.runtime(ctx, tenantInfo, agentextension.TypeExa, true)
	if err != nil {
		return agentextension.ExaSettings{}, err
	}

	settings, err := agentextension.ParseExaSettings(runtime.values)
	if err != nil {
		s.l.Error("stored web search settings are invalid", zap.Error(err))
		return agentextension.ExaSettings{}, &ToolError{Message: invalidSettingsMessage}
	}

	return settings, nil
}

func (s *Service) providerError(err error, action string) error {
	var message string
	switch {
	case exa.IsUnauthorized(err):
		message = providerExa + " rejected the organization's API key, so it could not " + action +
			". An administrator can update the key under AI Control, Extensions."
	case exa.IsOutOfCredits(err):
		message = "The organization's " + providerExa + " account is out of credits, so it could not " +
			action + ". An administrator can add credits in the " + providerExa + " dashboard."
	case exa.IsRateLimited(err):
		message = providerExa + " is limiting how fast requests can be made. Wait a minute before " +
			"trying again."
	case exa.IsInvalidRequest(err):
		message = providerExa + " could not " + action + " as asked"
		var apiErr *restx.APIError
		if errors.As(err, &apiErr) && apiErr.Message != "" {
			message += ": " + apiErr.Message
		}
		message += ". Rephrase the request."
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		message = providerExa + " took too long to " + action + ". Try again, or answer without it."
	default:
		s.l.Warn("web research provider failed", zap.Error(err), zap.String("action", action))
		message = providerExa + " could not be reached to " + action + ". Try again shortly, or " +
			"answer without it and say that the web could not be checked."
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

func dailyLimitMessage(limit int) string {
	return "This organization has used its daily limit of " + strconv.Itoa(limit) +
		" web requests. It resets at midnight UTC. Answer from what you already have and say " +
		"that the web could not be checked again today."
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
