/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package email

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Log)(nil)
	_ infra.PostgresSearchable  = (*Log)(nil)
)

// WebhookEvent represents a webhook event from email provider
type WebhookEvent struct {
	Provider  string         `json:"provider"`
	EventType string         `json:"eventType"`
	Timestamp int64          `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// Log represents an email delivery log entry
type Log struct {
	bun.BaseModel `bun:"table:email_logs,alias:el" json:"-"`

	// Primary identifiers
	ID               pulid.ID       `json:"id"                         bun:"id,type:varchar(100),pk"`
	OrganizationID   pulid.ID       `json:"organizationId"             bun:"organization_id,type:varchar(100),notnull"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"             bun:"business_unit_id,type:varchar(100),notnull"`
	QueueID          pulid.ID       `json:"queueId"                    bun:"queue_id,type:varchar(100),notnull"`
	MessageID        string         `json:"messageId"                  bun:"message_id,type:varchar(100)"` // Provider message ID
	Status           LogStatus      `json:"status"                     bun:"status,type:email_log_status_enum,notnull"`
	ProviderResponse string         `json:"providerResponse,omitempty" bun:"provider_response,type:text"`
	OpenedAt         *int64         `json:"openedAt,omitempty"         bun:"opened_at,type:bigint"`
	ClickedAt        *int64         `json:"clickedAt,omitempty"        bun:"clicked_at,type:bigint"`
	BouncedAt        *int64         `json:"bouncedAt,omitempty"        bun:"bounced_at,type:bigint"`
	ComplainedAt     *int64         `json:"complainedAt,omitempty"     bun:"complained_at,type:bigint"`
	UnsubscribedAt   *int64         `json:"unsubscribedAt,omitempty"   bun:"unsubscribed_at,type:bigint"`
	BounceType       *BounceType    `json:"bounceType,omitempty"       bun:"bounce_type,type:email_bounce_type_enum"`
	BounceReason     string         `json:"bounceReason,omitempty"     bun:"bounce_reason,type:text"`
	WebhookEvents    []WebhookEvent `json:"webhookEvents,omitempty"    bun:"webhook_events,type:jsonb"`
	UserAgent        string         `json:"userAgent,omitempty"        bun:"user_agent,type:text"`
	IPAddress        string         `json:"ipAddress,omitempty"        bun:"ip_address,type:inet"`
	ClickedURLs      []string       `json:"clickedUrls,omitempty"      bun:"clicked_urls,type:text[]"`
	Metadata         map[string]any `json:"metadata,omitempty"         bun:"metadata,type:jsonb"`
	CreatedAt        int64          `json:"createdAt"                  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Queue        *Queue                     `json:"queue"                  bun:"rel:belongs-to,join:queue_id=id"`
}

// HasOpened returns true if the email has been opened
func (l *Log) HasOpened() bool {
	return l.OpenedAt != nil && *l.OpenedAt > 0
}

// HasClicked returns true if any link in the email has been clicked
func (l *Log) HasClicked() bool {
	return l.ClickedAt != nil && *l.ClickedAt > 0
}

// HasBounced returns true if the email bounced
func (l *Log) HasBounced() bool {
	return l.BouncedAt != nil && *l.BouncedAt > 0
}

// IsHardBounce returns true if this is a hard bounce
func (l *Log) IsHardBounce() bool {
	return l.BounceType != nil && *l.BounceType == BounceTypeHard
}

// GetEngagementScore calculates a simple engagement score
func (l *Log) GetEngagementScore() int {
	score := 0

	if l.Status == LogStatusDelivered {
		score += 1
	}

	if l.HasOpened() {
		score += 2
	}

	if l.HasClicked() {
		score += 3
	}

	if l.Status == LogStatusComplained || l.Status == LogStatusUnsubscribed {
		score = 0
	}

	return score
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface
func (l *Log) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("eml_") // email log
		}
		l.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (l *Log) GetTableName() string {
	return "email_logs"
}

// GetPostgresSearchConfig implements the PostgresSearchable interface
func (l *Log) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "el",
		Fields: []infra.PostgresSearchableField{
			{Name: "message_id", Weight: "A", Type: infra.PostgresSearchTypeText},
			{Name: "provider_response", Weight: "B", Type: infra.PostgresSearchTypeText},
			{Name: "status", Weight: "C", Type: infra.PostgresSearchTypeEnum},
			{Name: "bounce_reason", Weight: "D", Type: infra.PostgresSearchTypeText},
		},
		MinLength:       2,
		MaxTerms:        5,
		UsePartialMatch: true,
	}
}
