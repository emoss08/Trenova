package billingqueueassist

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	searchMaxRunes  = 300
	searchMaxTokens = 500
	secondsPerDay   = 24 * 60 * 60
	searchMaxDays   = 366
	searchDayLayout = "2006-01-02"
)

const searchSystem = `You turn a biller's sentence into filters for the billing queue.

Lanes: Checking (being checked), NeedsYou (a check needs the biller), Waiting (waiting on a driver,
approver or credit), Ready (ready to approve), Held (on hold or sent back), Approved (approved, not posted).
Return only what the sentence asks for:
- lanes: the lanes it names, or none.
- payer: the customer it names, as written, or "". A customer name on its own is the payer: the
  customer the bill goes to.
- customer: the customer whose shipments it names, as written, only when the sentence is about the
  shipments themselves ("shipments for", "loads we hauled for", "shipped by"), or "".
- search: a PRO, BOL, PO or reference number it names, or "".
- minAmount / maxAmount: amounts in dollars, or null.
- olderThanDays: "older than N days" or "waiting N days" as a whole number, or null.
- delivered, pickedUp, queued: when the shipments were delivered, when they were picked up, and
  when the items entered the queue. For each, lastDays for "the last N days", "past week" (7) or
  "past month" (30); or from and to as YYYY-MM-DD, both days included, for named days and ranges
  ("yesterday", "last week" is the Monday to Sunday before this week, "in October"), counted from
  today's date, which you are given. A date the sentence gives without saying which ("this week")
  is when the shipments were delivered. Leave all three null for a date the sentence does not name.
- unresolved: any part of the sentence you could not turn into a filter, or "".`

type searchAnswer struct {
	Lanes         []string          `json:"lanes"`
	Payer         string            `json:"payer"`
	Customer      string            `json:"customer"`
	Search        string            `json:"search"`
	MinAmount     *float64          `json:"minAmount"`
	MaxAmount     *float64          `json:"maxAmount"`
	OlderThanDays *int              `json:"olderThanDays"`
	Delivered     searchDatesAnswer `json:"delivered"`
	PickedUp      searchDatesAnswer `json:"pickedUp"`
	Queued        searchDatesAnswer `json:"queued"`
	Unresolved    string            `json:"unresolved"`
}

type searchDatesAnswer struct {
	LastDays *int    `json:"lastDays"`
	From     *string `json:"from"`
	To       *string `json:"to"`
}

func (s *Service) ComposeSearch(
	ctx context.Context,
	req *billingqueueports.SearchIntentRequest,
) (*billingqueueports.SearchIntent, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" || len([]rune(text)) > searchMaxRunes {
		return nil, errortypes.NewValidationError(
			"text",
			errortypes.ErrInvalid,
			"Ask in a sentence of up to {0} characters",
			searchMaxRunes,
		)
	}
	available, err := s.canServe(ctx, aitask.TaskQueryCompose, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, errortypes.NewBusinessError("Searching in a sentence needs an AI provider")
	}

	today := searchToday(req.Today)
	var answer searchAnswer
	if _, err = s.nova.Run(ctx, searchTask(req, text, today), &answer); err != nil {
		return nil, searchFailure(err)
	}

	return s.resolveIntent(ctx, req.TenantInfo, &answer, today)
}

// searchToday is the person's date, or the server's when they sent none it can read.
func searchToday(raw string) time.Time {
	if day, err := time.Parse(searchDayLayout, strings.TrimSpace(raw)); err == nil {
		return day
	}
	now := time.Unix(timeutils.NowUnix(), 0).UTC()

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// resolveDates is the window an answer names, or nil with the part it could not
// read: a range needs its first day (an open start reads to today), and every
// window stays within a year of today.
func resolveDates(answer searchDatesAnswer, today time.Time) (*billingqueueports.SearchDates, string) {
	if answer.LastDays != nil {
		if days := *answer.LastDays; days > 0 && days <= searchMaxDays {
			return &billingqueueports.SearchDates{LastDays: days}, ""
		}
		return nil, ""
	}
	from := strings.TrimSpace(deref(answer.From))
	to := strings.TrimSpace(deref(answer.To))
	if from == "" && to == "" {
		return nil, ""
	}
	start, err := time.Parse(searchDayLayout, from)
	if err != nil {
		return nil, strings.TrimSpace(from + " " + to)
	}
	end := today
	if to != "" {
		if end, err = time.Parse(searchDayLayout, to); err != nil {
			return nil, strings.TrimSpace(from + " " + to)
		}
	}
	if end.Before(start) || today.Sub(start) > searchMaxDays*24*time.Hour ||
		end.Sub(today) > searchMaxDays*24*time.Hour {
		return nil, strings.TrimSpace(from + " " + to)
	}

	return &billingqueueports.SearchDates{
		From: start.Format(searchDayLayout),
		To:   end.Format(searchDayLayout),
	}, ""
}

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func (s *Service) resolveIntent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	answer *searchAnswer,
	today time.Time,
) (*billingqueueports.SearchIntent, error) {
	intent := &billingqueueports.SearchIntent{
		Lanes:      make([]billingqueue.Lane, 0, len(answer.Lanes)),
		Search:     strings.TrimSpace(answer.Search),
		Unresolved: strings.TrimSpace(answer.Unresolved),
	}
	for _, raw := range answer.Lanes {
		if lane := billingqueue.Lane(raw); lane.IsValid() && lane != billingqueue.LaneClosed {
			intent.Lanes = append(intent.Lanes, lane)
		}
	}
	if answer.MinAmount != nil && *answer.MinAmount > 0 {
		intent.MinAmount = decimal.NewNullDecimal(decimal.NewFromFloat(*answer.MinAmount).Round(2))
	}
	if answer.MaxAmount != nil && *answer.MaxAmount > 0 {
		intent.MaxAmount = decimal.NewNullDecimal(decimal.NewFromFloat(*answer.MaxAmount).Round(2))
	}
	if answer.OlderThanDays != nil && *answer.OlderThanDays > 0 {
		before := timeutils.NowUnix() - int64(*answer.OlderThanDays)*secondsPerDay
		intent.CreatedBefore = &before
	}
	unread := []string{intent.Unresolved}
	var missed string
	intent.Delivered, missed = resolveDates(answer.Delivered, today)
	unread = append(unread, missed)
	intent.PickedUp, missed = resolveDates(answer.PickedUp, today)
	unread = append(unread, missed)
	intent.Queued, missed = resolveDates(answer.Queued, today)
	unread = append(unread, missed)

	payer, err := s.findCustomer(ctx, tenantInfo, answer.Payer)
	if err != nil {
		return nil, err
	}
	if payer.found {
		intent.PayerID, intent.PayerName = payer.id, payer.name
	}
	customer, err := s.findCustomer(ctx, tenantInfo, answer.Customer)
	if err != nil {
		return nil, err
	}
	if customer.found {
		intent.ShipmentCustomerID, intent.ShipmentCustomerName = customer.id, customer.name
	}
	unread = append(unread, payer.missed, customer.missed)
	intent.Unresolved = joinUnread(unread)

	return intent, nil
}

type customerMatch struct {
	id     pulid.ID
	name   string
	found  bool
	missed string
}

// findCustomer is the customer a sentence names, the best match by name or code;
// a name that matches none is returned as what could not be read.
func (s *Service) findCustomer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	named string,
) (customerMatch, error) {
	named = strings.TrimSpace(named)
	if named == "" || s.customers == nil {
		return customerMatch{}, nil
	}
	matches, err := s.customers.SelectOptions(ctx, &customerports.CustomerSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: 1},
			Query:      named,
		},
	})
	if err != nil {
		return customerMatch{}, err
	}
	if matches == nil || len(matches.Items) == 0 {
		return customerMatch{missed: named}, nil
	}

	return customerMatch{id: matches.Items[0].ID, name: matches.Items[0].Name, found: true}, nil
}

func joinUnread(parts []string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			kept = append(kept, part)
		}
	}

	return strings.Join(kept, " ")
}

func searchTask(
	req *billingqueueports.SearchIntentRequest,
	text string,
	today time.Time,
) *llmports.NovaTask {
	nullableNumber := map[string]any{"type": []string{"number", "null"}}
	nullableDay := map[string]any{"type": []string{"string", "null"}}
	dates := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"lastDays": map[string]any{"type": []string{"integer", "null"}},
			"from":     nullableDay,
			"to":       nullableDay,
		},
		"required":             []string{"lastDays", "from", "to"},
		"additionalProperties": false,
	}

	return &llmports.NovaTask{
		Name:         aitask.TaskQueryCompose,
		Feature:      aiusage.FeatureBillingQueueSearch,
		Instructions: searchSystem,
		SchemaName:   "billing_queue_search",
		TenantInfo:   req.TenantInfo,
		UserID:       req.UserID,
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"lanes": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
						"enum": []string{
							"Checking",
							"NeedsYou",
							"Waiting",
							"Ready",
							"Held",
							"Approved",
						},
					},
				},
				"payer":         map[string]any{"type": "string"},
				"customer":      map[string]any{"type": "string"},
				"search":        map[string]any{"type": "string"},
				"minAmount":     nullableNumber,
				"maxAmount":     nullableNumber,
				"olderThanDays": map[string]any{"type": []string{"integer", "null"}},
				"delivered":     dates,
				"pickedUp":      dates,
				"queued":        dates,
				"unresolved":    map[string]any{"type": "string"},
			},
			"required": []string{
				"lanes", "payer", "customer", "search", "minAmount", "maxAmount", "olderThanDays",
				"delivered", "pickedUp", "queued", "unresolved",
			},
			"additionalProperties": false,
		},
		MaxTokens: searchMaxTokens,
		Context: llmports.DelimitedContext{Sections: []llmports.ContextSection{
			{
				Title:   "Today's date",
				Trusted: true,
				Content: today.Format(searchDayLayout) + " (" + today.Weekday().String() + ")",
			},
			{Title: "The biller's sentence", Content: text},
		}},
	}
}

// searchFailure says what went wrong in the words the person can act on: a
// sentence Nova could not turn into filters is worth rewording; a missing or
// unreachable provider is not.
func searchFailure(err error) error {
	var failed *llmports.ChatProvidersFailedError
	switch {
	case errors.Is(err, llmports.ErrOutsideSchema):
		return errortypes.NewBusinessError("Nova couldn't read that; try different words").
			WithInternal(err)
	case errors.Is(err, llmports.ErrNoProviderConfigured):
		return errortypes.NewBusinessError(
			"No AI provider reads queue searches. An administrator can assign one in AI Control.",
		).WithInternal(err)
	case errortypes.IsBusinessError(err):
		return err
	case errors.As(err, &failed):
		return errortypes.NewBusinessError(
			"Nova couldn't be reached just now. Try again in a moment.",
		).WithInternal(err)
	default:
		return errortypes.NewBusinessError("Nova couldn't search the queue just now. Try again.").
			WithInternal(err)
	}
}
