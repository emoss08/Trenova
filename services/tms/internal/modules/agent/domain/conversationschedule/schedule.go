// Package conversationschedule is a request a person asked the Desk to repeat
// on a cadence, such as "every weekday at 7:30, what's blocking the billing
// queue?", and the answers that come back as turns in the conversation it was
// asked in.
package conversationschedule

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
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
	keel.Table `keel:"conversation_schedules"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:csch_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ThreadID pulid.ID `json:"threadId" keel:"thread_id"`
	UserID   pulid.ID `json:"userId" keel:"user_id"`

	// Prompt is what each run asks, and Cadence when, as the person reads
	// it: "Every weekday · 7:30 AM". CronExpression is the same cadence for
	// the scheduler, read on Timezone's clock.
	Prompt         string `json:"prompt" keel:"prompt"`
	Cadence        string `json:"cadence" keel:"cadence"`
	CronExpression string `json:"cronExpression" keel:"cron_expression"`
	Timezone       string `json:"timezone" keel:"timezone"`

	// Enabled is false while the schedule is paused.
	Enabled bool `json:"enabled" keel:"enabled"`

	// LastRunAt is when the latest run started, and LastTurnID the turn it
	// started. NextRunAt is the next slot, worked out whenever the schedule
	// is saved or runs; it is kept while paused so resuming need not wait on
	// a read to show it.
	LastRunAt  *int64   `json:"lastRunAt" keel:"last_run_at"`
	NextRunAt  *int64   `json:"nextRunAt" keel:"next_run_at"`
	LastTurnID pulid.ID `json:"lastTurnId" keel:"last_turn_id,nullzero"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (s *Schedule) BeforeCreate(_ context.Context) error {
	if s.Timezone == "" {
		s.Timezone = DefaultTimezone
	}
	if s.Timezone == "" {
		s.Timezone = "UTC"
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
