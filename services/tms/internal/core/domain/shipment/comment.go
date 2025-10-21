package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

//nolint:revive // valid struct name
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
	MentionedUser *tenant.User         `json:"mentionedUser" bun:"rel:belongs-to,join:mentioned_user_id=id"`
	Comment       *ShipmentComment     `json:"-"             bun:"rel:belongs-to,join:comment_id=id"`
	Organization  *tenant.Organization `json:"-"             bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit  *tenant.BusinessUnit `json:"-"             bun:"rel:belongs-to,join:business_unit_id=id"`
}

func (scm *ShipmentCommentMention) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if scm.ID.IsNil() {
			scm.ID = pulid.MustNew("scm_")
		}
		scm.CreatedAt = utils.NowUnix()
	}
	return nil
}

var (
	_ bun.BeforeAppendModelHook = (*ShipmentComment)(nil)
	_ domain.Validatable        = (*ShipmentComment)(nil)
	_ framework.TenantedEntity  = (*ShipmentComment)(nil)
)

//nolint:revive // valid struct name
type ShipmentComment struct {
	bun.BaseModel `bun:"table:shipment_comments,alias:sc" json:"-"`

	ID             pulid.ID                  `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID                  `json:"businessUnitId"     bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID                  `json:"organizationId"     bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ShipmentID     pulid.ID                  `json:"shipmentId"         bun:"shipment_id,pk,notnull,type:VARCHAR(100)"`
	UserID         pulid.ID                  `json:"userId"             bun:"user_id,pk,notnull,type:VARCHAR(100)"`
	Comment        string                    `json:"comment"            bun:"comment,type:TEXT,notnull"`
	CommentType    string                    `json:"commentType"        bun:"comment_type,type:VARCHAR(100)"`
	Metadata       map[string]any            `json:"metadata,omitempty" bun:"metadata,type:JSONB,default:'{}'::jsonb"`
	Version        int64                     `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt      int64                     `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64                     `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Shipment       *Shipment                 `json:"-"                  bun:"rel:belongs-to,join:shipment_id=id"`
	BusinessUnit   *tenant.BusinessUnit      `json:"-"                  bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization   *tenant.Organization      `json:"-"                  bun:"rel:belongs-to,join:organization_id=id"`
	User           *tenant.User              `json:"user"               bun:"rel:belongs-to,join:user_id=id"`
	MentionedUsers []*ShipmentCommentMention `json:"mentionedUsers"     bun:"rel:has-many,join:id=comment_id"`
}

func (sc *ShipmentComment) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(sc,
		validation.Field(&sc.Comment, validation.Required.Error("Comment is required")),
		validation.Field(&sc.Metadata, validation.Required.Error("Metadata is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (sc *ShipmentComment) GetID() string {
	return sc.ID.String()
}

func (sc *ShipmentComment) GetOrganizationID() pulid.ID {
	return sc.OrganizationID
}

func (sc *ShipmentComment) GetBusinessUnitID() pulid.ID {
	return sc.BusinessUnitID
}

func (sc *ShipmentComment) GetTableName() string {
	return "shipment_comments"
}

func (sc *ShipmentComment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

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
