// Package briefingwriter turns the morning's figures into the sentences a
// person reads.
//
// It is the only part of the briefing that talks to a model, and it is the
// least trusted part. It cannot add a section, a figure or a link: it is
// given the page as computed and returns a headline and one sentence per
// section. Every sentence is checked against the figures that were gathered,
// and anything citing a number nobody computed is dropped in favour of the
// deterministic wording. A briefing without a model is a complete briefing.
package briefingwriter

import (
	"context"
	"errors"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// maxOutputTokens bounds one briefing. A headline and a sentence per
// section cannot need more, and an unbounded ceiling on a job that runs per
// organization per role is how a morning quietly becomes expensive.
const maxOutputTokens = 900

type Params struct {
	fx.In

	Logger     *zap.Logger
	Completion services.CompletionService
}

type Service struct {
	l          *zap.Logger
	completion services.CompletionService
}

func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.briefing-writer"),
		completion: p.Completion,
	}
}

// Request is one role's page, as computed, with the figures it may cite.
type Request struct {
	TenantInfo pagination.TenantInfo
	Role       briefing.RoleKey
	// OrganizationName is who the briefing is for, so the headline reads
	// like a sentence about a company rather than about a database.
	OrganizationName string
	Sections         []briefing.Section
	Supported        []decimal.Decimal
}

// Result is the wording, and whether any of it was the model's.
type Result struct {
	Headline string
	// Bodies is the model's sentence per section key, only for the
	// sections whose wording survived the guard.
	Bodies map[briefing.SectionKey]string
	// Narrated is false when no model wrote any of it: unavailable,
	// unparseable, or overruled on every section.
	Narrated        bool
	ModelIdentifier string
	ProviderID      pulid.ID
}

// Write asks a model for the page's wording. It never returns an error: a
// briefing whose prose could not be written is still a briefing, and the
// caller keeps the deterministic summaries.
func (s *Service) Write(ctx context.Context, req *Request) Result {
	result := Result{
		Headline: fallbackHeadline(req.Sections),
		Bodies:   map[briefing.SectionKey]string{},
	}
	if len(req.Sections) == 0 {
		return result
	}

	completion, err := s.completion.CompleteStructured(ctx, &services.StructuredCompletionRequest{
		TenantInfo:   req.TenantInfo,
		Task:         aiprovider.TaskDailyBriefing,
		System:       systemPrompt,
		Context:      buildContext(req),
		OutputSchema: outputSchema(),
		SchemaName:   "daily_briefing",
		MaxTokens:    maxOutputTokens,
	})
	if err != nil {
		s.logUnavailable(err, req.Role)

		return result
	}

	written, err := parse(completion.Text)
	if err != nil {
		s.l.Warn("briefing wording could not be parsed",
			zap.String("model", completion.ModelIdentifier),
			zap.String("role", string(req.Role)),
			zap.Error(err),
		)

		return result
	}

	s.apply(req, written, completion, &result)

	return result
}

// apply accepts the model's wording only where it survives the guard. The
// check runs per sentence rather than per page so one invented figure costs
// that section its prose and no other.
func (s *Service) apply(
	req *Request,
	written *briefingDraft,
	completion *services.StructuredCompletionResult,
	result *Result,
) {
	known := make(map[briefing.SectionKey]struct{}, len(req.Sections))
	for _, section := range req.Sections {
		known[section.Key] = struct{}{}
	}

	if headline := strings.TrimSpace(written.Headline); headline != "" {
		if check := numberguard.CheckNumbers(headline, req.Supported); check.OK {
			result.Headline = headline
			result.Narrated = true
		} else {
			s.rejected(req, completion, "headline", check.Unsupported)
		}
	}

	for _, section := range written.Sections {
		key := briefing.SectionKey(section.Key)
		if _, ok := known[key]; !ok {
			// A section nobody computed is not a section: the page's shape
			// is decided before the model runs.
			continue
		}
		body := strings.TrimSpace(section.Body)
		if body == "" {
			continue
		}
		if check := numberguard.CheckNumbers(body, req.Supported); !check.OK {
			s.rejected(req, completion, section.Key, check.Unsupported)

			continue
		}
		result.Bodies[key] = body
		result.Narrated = true
	}

	if result.Narrated {
		result.ModelIdentifier = completion.ModelIdentifier
		result.ProviderID = completion.ProviderID
	}
}

func (s *Service) rejected(
	req *Request,
	completion *services.StructuredCompletionResult,
	where string,
	unsupported []string,
) {
	s.l.Warn("rejected briefing wording citing figures nobody computed",
		zap.String("role", string(req.Role)),
		zap.String("section", where),
		zap.String("model", completion.ModelIdentifier),
		zap.Strings("unsupported", unsupported),
	)
}

func (s *Service) logUnavailable(err error, role briefing.RoleKey) {
	if errors.Is(err, services.ErrNoProviderConfigured) {
		s.l.Info("no provider configured for the daily briefing; using computed wording",
			zap.String("role", string(role)),
		)

		return
	}

	s.l.Warn("briefing wording failed; using computed wording",
		zap.String("role", string(role)),
		zap.Error(err),
	)
}

type briefingDraft struct {
	Headline string `json:"headline"`
	Sections []struct {
		Key  string `json:"key"`
		Body string `json:"body"`
	} `json:"sections"`
}

func parse(text string) (*briefingDraft, error) {
	draft := new(briefingDraft)
	if err := sonic.Unmarshal([]byte(text), draft); err != nil {
		return nil, err
	}

	return draft, nil
}

// fallbackHeadline is what the page says when no model wrote it: the first
// computed summary, which is plainer and always true.
func fallbackHeadline(sections []briefing.Section) string {
	for _, section := range sections {
		if summary := strings.TrimSpace(section.Summary); summary != "" {
			return summary
		}
	}

	return "Here is your day."
}
