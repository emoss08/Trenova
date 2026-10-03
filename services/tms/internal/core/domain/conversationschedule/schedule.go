// Package conversationschedule is a request a person asked the Desk to repeat
// on a cadence, such as "every weekday at 7:30, what's blocking the billing
// queue?", and the answers that come back as turns in the conversation it was
// asked in.
package conversationschedule

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Schedule)(nil)
	_ validationframework.TenantedEntity = (*Schedule)(nil)
)

const (
	// MaxPromptLength bounds what one run asks. It is a request somebody
	// typed, not a document.
	MaxPromptLength = 2000

	// MaxPerThread and MaxPerUser bound how many schedules one conversation
	// and one person keep. Each is a Temporal schedule and a model turn on
	// every slot, and a conversation's cards are read back in one page.
	MaxPerThread = 25
	MaxPerUser   = 100

	// DefaultTimezone is the clock a schedule is read on when neither the
	// person nor their organization names one.
	DefaultTimezone = "UTC"
)

// Schedule is one repeated request in one person's conversation.
//
// It belongs to the person who asked it, like the conversation does. Every
// run is a turn in that conversation, asked with their access at the time it
// runs, so a person who loses access to the agent stops getting answers
// rather than getting them with access they no longer hold.
type Schedule struct {
	bun.BaseModel `bun:"table:conversation_schedules,alias:csch" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	ThreadID pulid.ID `json:"threadId" bun:"thread_id,type:VARCHAR(100),notnull"`
	UserID   pulid.ID `json:"userId"   bun:"user_id,type:VARCHAR(100),notnull"`

	// Prompt is what each run asks, and Cadence when, as the person reads
	// it: "Every weekday · 7:30 AM". CronExpression is the same cadence for
	// the scheduler, read on Timezone's clock.
	Prompt         string `json:"prompt"         bun:"prompt,type:TEXT,notnull"`
	Cadence        string `json:"cadence"        bun:"cadence,type:VARCHAR(100),notnull"`
	CronExpression string `json:"cronExpression" bun:"cron_expression,type:VARCHAR(100),notnull"`
	Timezone       string `json:"timezone"       bun:"timezone,type:VARCHAR(100),notnull,default:'UTC'"`

	// Enabled is false while the schedule is paused.
	Enabled bool `json:"enabled" bun:"enabled,type:BOOLEAN,notnull,default:true"`

	// LastRunAt is when the latest run started, and LastTurnID the turn it
	// started. NextRunAt is the next slot, worked out whenever the schedule
	// is saved or runs; it is kept while paused so resuming need not wait on
	// a read to show it.
	LastRunAt  *int64   `json:"lastRunAt"  bun:"last_run_at,type:BIGINT,nullzero"`
	NextRunAt  *int64   `json:"nextRunAt"  bun:"next_run_at,type:BIGINT,nullzero"`
	LastTurnID pulid.ID `json:"lastTurnId" bun:"last_turn_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
}

func (s *Schedule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("csch_")
		}
		if s.Timezone == "" {
			s.Timezone = DefaultTimezone
		}
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}

func (s *Schedule) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&s.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&s.ThreadID, validation.Required.Error("Conversation is required")),
		validation.Field(&s.UserID, validation.Required.Error("User is required")),
		validation.Field(&s.Prompt,
			validation.Required.Error("Say what to ask on each run"),
			validation.RuneLength(1, MaxPromptLength).
				Error("A scheduled request cannot be longer than 2000 characters"),
		),
		validation.Field(&s.Cadence, validation.Required.Error("Cadence is required")),
		validation.Field(&s.CronExpression,
			validation.Required.Error("Cadence is required"),
			validation.By(func(any) error { return cronutils.Validate(s.CronExpression) }),
		),
		validation.Field(&s.Timezone,
			validation.Required.Error("Timezone is required"),
			validation.By(func(any) error {
				_, err := time.LoadLocation(s.Timezone)
				return err
			}),
		),
	))
}

// Next is the schedule's first slot after the instant given, or nil when its
// cadence cannot be read.
func (s *Schedule) Next(after int64) *int64 {
	next, err := cronutils.NextRun(s.CronExpression, s.Timezone, after)
	if err != nil {
		return nil
	}

	return &next
}

func (s *Schedule) GetID() pulid.ID { return s.ID }

func (s *Schedule) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *Schedule) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *Schedule) GetTableName() string { return "conversation_schedules" }
