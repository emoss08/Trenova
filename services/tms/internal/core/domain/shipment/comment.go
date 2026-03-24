package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	MaxCommentLength   = 5000
	MaxCommentMentions = 20
)

type ShipmentComment struct {
	bun.BaseModel `bun:"table:shipment_comments,alias:sc" json:"-"`

	ID               pulid.ID          `json:"id"                         bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID          `json:"businessUnitId"             bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID          `json:"organizationId"             bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID       pulid.ID          `json:"shipmentId"                 bun:"shipment_id,pk,type:VARCHAR(100),notnull"`
	UserID           pulid.ID          `json:"userId"                     bun:"user_id,type:VARCHAR(100),notnull"`
	Comment          string            `json:"comment"                    bun:"comment,type:TEXT,notnull"`
	Type             CommentType       `json:"type"                       bun:"type,type:VARCHAR(50),notnull,default:'Internal'"`
	Visibility       CommentVisibility `json:"visibility"                 bun:"visibility,type:VARCHAR(50),notnull,default:'Internal'"`
	Priority         CommentPriority   `json:"priority"                   bun:"priority,type:VARCHAR(20),notnull,default:'Normal'"`
	Source           CommentSource     `json:"source"                     bun:"source,type:VARCHAR(20),notnull,default:'User'"`
	Metadata         map[string]any    `json:"metadata,omitempty"         bun:"metadata,type:JSONB,default:'{}'::jsonb"`
	EditedAt         *int64            `json:"editedAt"                   bun:"edited_at,type:BIGINT,nullzero"`
	Version          int64             `json:"version"                    bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt        int64             `json:"createdAt"                  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64             `json:"updatedAt"                  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	MentionedUserIDs []pulid.ID        `json:"mentionedUserIds,omitempty" bun:"-"`

	User           *tenant.User              `json:"user,omitempty"           bun:"rel:belongs-to,join:user_id=id"`
	MentionedUsers []*ShipmentCommentMention `json:"mentionedUsers,omitempty" bun:"rel:has-many,join:id=comment_id"`
}

type ShipmentCommentMention struct {
	bun.BaseModel `bun:"table:shipment_comment_mentions,alias:scm" json:"-"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	CommentID       pulid.ID `json:"commentId"       bun:"comment_id,type:VARCHAR(100),notnull"`
	MentionedUserID pulid.ID `json:"mentionedUserId" bun:"mentioned_user_id,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),notnull"`
	ShipmentID      pulid.ID `json:"shipmentId"      bun:"shipment_id,type:VARCHAR(100),notnull"`
	CreatedAt       int64    `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	MentionedUser *tenant.User `json:"mentionedUser,omitempty" bun:"rel:belongs-to,join:mentioned_user_id=id"`
}

func (c *ShipmentComment) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		c,
		validation.Field(
			&c.ShipmentID,
			validation.Required.Error("Shipment ID is required"),
		),
		validation.Field(
			&c.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&c.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&c.UserID,
			validation.Required.Error("User ID is required"),
		),
		validation.Field(
			&c.Comment,
			validation.Required.Error("Comment is required"),
			validation.Length(1, MaxCommentLength).
				Error("Comment must be between 1 and 5000 characters"),
		),
		validation.Field(
			&c.MentionedUserIDs,
			validation.By(func(value any) error {
				ids, _ := value.([]pulid.ID)
				if len(ids) > MaxCommentMentions {
					return errors.New("Mention count must be 20 or fewer")
				}
				return nil
			}),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (c *ShipmentComment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("shc_")
		}
		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func (m *ShipmentCommentMention) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("scm_")
		}
		if m.CreatedAt == 0 {
			m.CreatedAt = timeutils.NowUnix()
		}
	}

	return nil
}

func (c *ShipmentComment) GetID() pulid.ID {
	return c.ID
}

func (c *ShipmentComment) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *ShipmentComment) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}
