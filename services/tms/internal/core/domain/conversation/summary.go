package conversation

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	MaxSummaryContentChars   = 16000
	MaxSummaryItems          = 16
	MaxSummaryItemChars      = 400
	MaxSummaryRecords        = 24
	MaxSummaryNarrativeChars = 2000
)

type SummaryTrigger string

const (
	SummaryTriggerThreshold = SummaryTrigger("Threshold")
	SummaryTriggerOverflow  = SummaryTrigger("Overflow")
)

func (t SummaryTrigger) IsValid() bool {
	switch t {
	case SummaryTriggerThreshold, SummaryTriggerOverflow:
		return true
	default:
		return false
	}
}

type SummaryRecord struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type SummarySections struct {
	Goal        string          `json:"goal"`
	State       string          `json:"state"`
	Decisions   []string        `json:"decisions,omitempty"`
	OpenItems   []string        `json:"openItems,omitempty"`
	Records     []SummaryRecord `json:"records,omitempty"`
	Facts       []string        `json:"facts,omitempty"`
	Preferences []string        `json:"preferences,omitempty"`
}

type Summary struct {
	bun.BaseModel `bun:"table:assistant_thread_summaries,alias:atsum" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	ThreadID           pulid.ID        `json:"threadId"           bun:"thread_id,type:VARCHAR(100),notnull"`
	FromSequence       int             `json:"fromSequence"       bun:"from_sequence,type:INTEGER,notnull"`
	ThroughSequence    int             `json:"throughSequence"    bun:"through_sequence,type:INTEGER,notnull"`
	Content            string          `json:"content"            bun:"content,type:TEXT,notnull"`
	Sections           SummarySections `json:"sections"           bun:"sections,type:JSONB,notnull"`
	Trigger            SummaryTrigger  `json:"trigger"            bun:"trigger,type:VARCHAR(20),notnull"`
	Tainted            bool            `json:"tainted"            bun:"tainted,type:BOOLEAN,notnull,default:false"`
	Taint              *agent.RunTaint `json:"taint,omitempty"    bun:"taint,type:JSONB,nullzero"`
	MessagesSummarized int             `json:"messagesSummarized" bun:"messages_summarized,type:INTEGER,notnull,default:0"`
	TokensBefore       int             `json:"tokensBefore"       bun:"tokens_before,type:INTEGER,notnull,default:0"`
	TokensAfter        int             `json:"tokensAfter"        bun:"tokens_after,type:INTEGER,notnull,default:0"`
	MemorySuggestions  int             `json:"memorySuggestions"  bun:"memory_suggestions,type:INTEGER,notnull,default:0"`

	ProviderID   pulid.ID         `json:"providerId,omitempty" bun:"provider_id,type:VARCHAR(100),nullzero"`
	Model        string           `json:"model,omitempty"      bun:"model,type:VARCHAR(200),nullzero"`
	InputTokens  int              `json:"inputTokens"          bun:"input_tokens,type:INTEGER,notnull,default:0"`
	OutputTokens int              `json:"outputTokens"         bun:"output_tokens,type:INTEGER,notnull,default:0"`
	CostUSD      *decimal.Decimal `json:"costUsd,omitempty"    bun:"cost_usd,type:NUMERIC(14,6),nullzero"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (s *Summary) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("atsum_")
		}
		if s.CreatedAt == 0 {
			s.CreatedAt = timeutils.NowUnix()
		}
	}

	return nil
}

func (s *Summary) GetID() pulid.ID { return s.ID }

func (s *Summary) GetTableName() string { return "assistant_thread_summaries" }

func (s *Summary) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.ThreadID, validation.Required.Error("Thread is required")),
		validation.Field(&s.FromSequence,
			validation.Min(0).Error("From sequence cannot be negative"),
		),
		validation.Field(&s.Content,
			validation.Required.Error("Content is required"),
			validation.RuneLength(1, MaxSummaryContentChars).
				Error("Content cannot be longer than 16000 characters"),
		),
		validation.Field(&s.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[SummaryTrigger]("Trigger is invalid"),
		),
	))

	if s.ThroughSequence < s.FromSequence {
		multiErr.Add("throughSequence", errortypes.ErrInvalid,
			"A summary cannot end before it starts")
	}
	if s.Tainted != s.Taint.Tainted() {
		multiErr.Add("tainted", errortypes.ErrInvalid,
			"A summary is tainted exactly when it carries the outside content it read")
	}
}

func (s *Summary) Covers(sequence int) bool {
	return s != nil && sequence <= s.ThroughSequence
}

func (s SummarySections) Bounded() SummarySections {
	return SummarySections{
		Goal:        boundedLine(s.Goal, MaxSummaryNarrativeChars),
		State:       boundedLine(s.State, MaxSummaryNarrativeChars),
		Decisions:   boundedItems(s.Decisions),
		OpenItems:   boundedItems(s.OpenItems),
		Records:     boundedRecords(s.Records),
		Facts:       boundedItems(s.Facts),
		Preferences: boundedItems(s.Preferences),
	}
}

func (s SummarySections) Empty() bool {
	return s.Goal == "" && s.State == "" && len(s.Decisions) == 0 &&
		len(s.OpenItems) == 0 && len(s.Records) == 0 && len(s.Facts) == 0 &&
		len(s.Preferences) == 0
}

func (s SummarySections) Render() string {
	var b strings.Builder
	b.Grow(1024)

	writeNarrative(&b, "What the person is working on", s.Goal)
	writeNarrative(&b, "Where the conversation stands", s.State)
	writeList(&b, "Decided or done", s.Decisions)
	writeList(&b, "Still open", s.OpenItems)
	if len(s.Records) > 0 {
		b.WriteString("### Records discussed\n")
		for _, record := range s.Records {
			b.WriteString("- ")
			b.WriteString(record.Kind)
			b.WriteString(" ")
			b.WriteString(record.ID)
			if record.Label != "" {
				b.WriteString(" (")
				b.WriteString(record.Label)
				b.WriteString(")")
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	writeList(&b, "Facts established", s.Facts)
	writeList(&b, "What the person asked for in how things are done", s.Preferences)

	return stringutils.Ellipsize(strings.TrimSpace(b.String()), MaxSummaryContentChars)
}

func writeNarrative(b *strings.Builder, heading, text string) {
	if text == "" {
		return
	}
	b.WriteString("### ")
	b.WriteString(heading)
	b.WriteString("\n")
	b.WriteString(text)
	b.WriteString("\n\n")
}

func writeList(b *strings.Builder, heading string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("### ")
	b.WriteString(heading)
	b.WriteString("\n")
	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func boundedLine(value string, limit int) string {
	return stringutils.Ellipsize(strings.Join(strings.Fields(value), " "), limit)
}

func boundedItems(items []string) []string {
	if len(items) == 0 {
		return nil
	}

	out := make([]string, 0, min(len(items), MaxSummaryItems))
	for _, item := range items {
		line := boundedLine(item, MaxSummaryItemChars)
		if line == "" {
			continue
		}
		out = append(out, line)
		if len(out) == MaxSummaryItems {
			break
		}
	}

	return out
}

func boundedRecords(records []SummaryRecord) []SummaryRecord {
	if len(records) == 0 {
		return nil
	}

	out := make([]SummaryRecord, 0, min(len(records), MaxSummaryRecords))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		kind := boundedLine(record.Kind, 60)
		id := boundedLine(record.ID, 100)
		if kind == "" || id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, SummaryRecord{
			Kind:  kind,
			ID:    id,
			Label: boundedLine(record.Label, 200),
		})
		if len(out) == MaxSummaryRecords {
			break
		}
	}

	return out
}
