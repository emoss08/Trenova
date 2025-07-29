/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipment

import (
	"context"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

// ShipmentCommentMention tracks users mentioned in a comment
type ShipmentCommentMention struct {
	bun.BaseModel `bun:"table:shipment_comment_mentions,alias:scm" json:"-"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	CommentID       pulid.ID `json:"commentId"       bun:"comment_id,notnull,type:VARCHAR(100)"`
	ShipmentID      pulid.ID `json:"shipmentId"      bun:"shipment_id,notnull,type:VARCHAR(100)"`
	MentionedUserID pulid.ID `json:"mentionedUserId" bun:"mentioned_user_id,notnull,type:VARCHAR(100)"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,notnull,type:VARCHAR(100)"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,notnull,type:VARCHAR(100)"`
	CreatedAt       int64    `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	MentionedUser *user.User                 `json:"mentionedUser" bun:"rel:belongs-to,join:mentioned_user_id=id"`
	Comment       *ShipmentComment           `json:"-"             bun:"rel:belongs-to,join:comment_id=id"`
	Organization  *organization.Organization `json:"-"             bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit  *businessunit.BusinessUnit `json:"-"             bun:"rel:belongs-to,join:business_unit_id=id"`
}

type ShipmentComment struct {
	bun.BaseModel `bun:"table:shipment_comments,alias:sc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,pk,notnull,type:VARCHAR(100)"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,pk,notnull,type:VARCHAR(100)"`
	Comment        string   `json:"comment"        bun:"comment,type:TEXT,notnull"`
	IsHighPriority bool     `json:"isHighPriority" bun:"is_high_priority,type:BOOLEAN,notnull,default:false"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	Shipment       *Shipment                  `json:"-"              bun:"rel:belongs-to,join:shipment_id=id"`
	BusinessUnit   *businessunit.BusinessUnit `json:"-"              bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization   *organization.Organization `json:"-"              bun:"rel:belongs-to,join:organization_id=id"`
	User           *user.User                 `json:"user"           bun:"rel:belongs-to,join:user_id=id"`
	MentionedUsers []*ShipmentCommentMention  `json:"mentionedUsers" bun:"rel:has-many,join:id=comment_id"`
}

func (sc *ShipmentComment) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, sc,
		validation.Field(&sc.Comment, validation.Required.Error("Comment is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (sc *ShipmentComment) GetID() string {
	return sc.ID.String()
}

func (sc *ShipmentComment) GetTableName() string {
	return "shipment_comments"
}

func (sc *ShipmentComment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if sc.ID.IsNil() {
			sc.ID = pulid.MustNew("sc_")
		}

		sc.CreatedAt = now
	case *bun.UpdateQuery:
		sc.UpdatedAt = now
	}

	return nil
}

// ExtractMentions extracts all @username mentions from the comment text
func (sc *ShipmentComment) ExtractMentions() []string {
	mentionRegex := regexp.MustCompile(`@(\w+)`)
	matches := mentionRegex.FindAllStringSubmatch(sc.Comment, -1)

	mentions := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			username := match[1]
			if !seen[username] {
				mentions = append(mentions, username)
				seen[username] = true
			}
		}
	}

	return mentions
}

// BeforeAppendModel for ShipmentCommentMention
func (scm *ShipmentCommentMention) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if scm.ID.IsNil() {
			scm.ID = pulid.MustNew("scm_")
		}
		scm.CreatedAt = timeutils.NowUnix()
	}
	return nil
}
