package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ pagination.CursorEntity        = (*ShipmentComment)(nil)
	_ domaintypes.PostgresSearchable = (*ShipmentComment)(nil)
)

const (
	MaxCommentLength      = 5000
	MaxCommentMentions    = 20
	MaxCommentAttachments = 10
	MaxPinnedComments     = 25
)

const CommentAttachmentResourceType = "shipment_comment"

type ShipmentComment struct {
	bun.BaseModel             `bun:"table:shipment_comments,alias:sc" json:"-"`
	pagination.CursorValueSet `bun:",embed"                           json:"-"`

	ID                     pulid.ID             `json:"id"                              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID             `json:"businessUnitId"                  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID             `json:"organizationId"                  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID             pulid.ID             `json:"shipmentId"                      bun:"shipment_id,pk,type:VARCHAR(100),notnull"`
	UserID                 pulid.ID             `json:"userId"                          bun:"user_id,type:VARCHAR(100),nullzero"`
	ParentCommentID        *pulid.ID            `json:"parentCommentId"                 bun:"parent_comment_id,type:VARCHAR(100),nullzero"`
	Comment                string               `json:"comment"                         bun:"comment,type:TEXT,notnull"`
	Body                   map[string]any       `json:"body,omitempty"                  bun:"body,type:JSONB,nullzero"`
	Type                   CommentType          `json:"type"                            bun:"type,type:VARCHAR(50),notnull,default:'Internal'"`
	Visibility             CommentVisibility    `json:"visibility"                      bun:"visibility,type:VARCHAR(50),notnull,default:'Internal'"`
	Priority               CommentPriority      `json:"priority"                        bun:"priority,type:VARCHAR(20),notnull,default:'Normal'"`
	Source                 CommentSource        `json:"source"                          bun:"source,type:VARCHAR(20),notnull,default:'User'"`
	Metadata               map[string]any       `json:"metadata,omitempty"              bun:"metadata,type:JSONB,default:'{}'"`
	EditedAt               *int64               `json:"editedAt"                        bun:"edited_at,type:BIGINT,nullzero"`
	PinnedAt               *int64               `json:"pinnedAt"                        bun:"pinned_at,type:BIGINT,nullzero"`
	PinnedByID             *pulid.ID            `json:"pinnedById"                      bun:"pinned_by_id,type:VARCHAR(100),nullzero"`
	ResolvedAt             *int64               `json:"resolvedAt"                      bun:"resolved_at,type:BIGINT,nullzero"`
	ResolvedByID           *pulid.ID            `json:"resolvedById"                    bun:"resolved_by_id,type:VARCHAR(100),nullzero"`
	RequiresAcknowledgment bool                 `json:"requiresAcknowledgment"          bun:"requires_acknowledgment,type:BOOLEAN,notnull,default:false"`
	DeletedAt              *int64               `json:"deletedAt"                       bun:"deleted_at,type:BIGINT,nullzero"`
	DeletedByID            *pulid.ID            `json:"deletedById"                     bun:"deleted_by_id,type:VARCHAR(100),nullzero"`
	SearchVector           string               `json:"-"                               bun:"search_vector,type:TSVECTOR,scanonly"`
	ReplyCount             int64                `json:"replyCount"                      bun:"reply_count,scanonly"`
	Version                int64                `json:"version"                         bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt              int64                `json:"createdAt"                       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64                `json:"updatedAt"                       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	MentionedUserIDs       []pulid.ID           `json:"mentionedUserIds,omitempty"      bun:"-"`
	AttachmentDocumentIDs  []pulid.ID           `json:"attachmentDocumentIds,omitempty" bun:"-"`
	Attachments            []*CommentAttachment `json:"attachments,omitempty"           bun:"-"`

	User            *tenant.User                     `json:"user,omitempty"            bun:"rel:belongs-to,join:user_id=id"`
	PinnedBy        *tenant.User                     `json:"pinnedBy,omitempty"        bun:"rel:belongs-to,join:pinned_by_id=id"`
	ResolvedBy      *tenant.User                     `json:"resolvedBy,omitempty"      bun:"rel:belongs-to,join:resolved_by_id=id"`
	MentionedUsers  []*ShipmentCommentMention        `json:"mentionedUsers,omitempty"  bun:"rel:has-many,join:id=comment_id"`
	Acknowledgments []*ShipmentCommentAcknowledgment `json:"acknowledgments,omitempty" bun:"rel:has-many,join:id=comment_id"`
}

type CommentAttachment struct {
	DocumentID   pulid.ID `json:"documentId"`
	FileName     string   `json:"fileName"`
	OriginalName string   `json:"originalName,omitempty"`
	FileSize     int64    `json:"fileSize"`
	MimeType     string   `json:"mimeType"`
	PreviewURL   string   `json:"previewUrl,omitempty"`
	DownloadURL  string   `json:"downloadUrl"`
	CreatedAt    int64    `json:"createdAt"`
}

type ShipmentCommentAcknowledgment struct {
	bun.BaseModel `bun:"table:shipment_comment_acknowledgments,alias:sca" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	CommentID      pulid.ID `json:"commentId"      bun:"comment_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	AcknowledgedAt int64    `json:"acknowledgedAt" bun:"acknowledged_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	User *tenant.User `json:"user,omitempty" bun:"rel:belongs-to,join:user_id=id"`
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
	if c.Source == "" {
		c.Source = CommentSourceUser
	}

	multiErr.AddOzzoError(validation.ValidateStruct(
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
			validation.When(
				c.Source != CommentSourceSystem,
				validation.Required.Error("User ID is required"),
			),
		),
		validation.Field(
			&c.Comment,
			validation.Required.Error("Comment is required"),
			validation.Length(1, MaxCommentLength).
				Error("Comment must be between 1 and 5000 characters"),
		),
		validation.Field(
			&c.Source,
			validation.In(
				CommentSourceUser,
				CommentSourceSystem,
				CommentSourceIntegration,
				CommentSourceAI,
			).Error("Source must be a valid comment source"),
		),
		validation.Field(
			&c.MentionedUserIDs,
			validation.By(func(value any) error {
				ids, _ := value.([]pulid.ID)
				if len(ids) > MaxCommentMentions {
					return errors.New("mention count must be 20 or fewer")
				}
				return nil
			}),
		),
		validation.Field(
			&c.AttachmentDocumentIDs,
			validation.By(func(value any) error {
				ids, _ := value.([]pulid.ID)
				if len(ids) > MaxCommentAttachments {
					return errors.New("attachment count must be 10 or fewer")
				}
				return nil
			}),
		),
	))
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

func (a *ShipmentCommentAcknowledgment) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("sck_")
		}
		if a.AcknowledgedAt == 0 {
			a.AcknowledgedAt = timeutils.NowUnix()
		}
	}

	return nil
}

func (c *ShipmentComment) IsReply() bool {
	return c.ParentCommentID != nil && !c.ParentCommentID.IsNil()
}

func (c *ShipmentComment) IsPinned() bool {
	return c.PinnedAt != nil
}

func (c *ShipmentComment) IsResolved() bool {
	return c.ResolvedAt != nil
}

func (c *ShipmentComment) IsDeleted() bool {
	return c.DeletedAt != nil
}

func (c *ShipmentComment) GetTableName() string {
	return "shipment_comments"
}

func (c *ShipmentComment) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "sc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "comment", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (c *ShipmentComment) GetID() pulid.ID {
	return c.ID
}

func (c *ShipmentComment) GetCreatedAt() int64 {
	return c.CreatedAt
}

func (c *ShipmentComment) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *ShipmentComment) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

func (a *ShipmentCommentAcknowledgment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&a.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&a.CommentID, validation.Required.Error("Comment is required")),
		validation.Field(&a.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&a.UserID, validation.Required.Error("User is required")),
		// The acknowledgment is the record that someone read a comment that
		// required reading, so an unstamped one proves nothing.
		validation.Field(&a.AcknowledgedAt,
			validation.Required.Error("Acknowledged at is required"),
			validation.Min(int64(1)).Error("Acknowledged at must be a valid timestamp"),
		),
	))
}

func (m *ShipmentCommentMention) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&m.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&m.CommentID, validation.Required.Error("Comment is required")),
		validation.Field(&m.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&m.MentionedUserID,
			validation.Required.Error("Mentioned user is required"),
		),
	))
}
