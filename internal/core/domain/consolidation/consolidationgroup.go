package consolidation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*ConsolidationGroup)(nil)
	_ infra.PostgresSearchable  = (*ConsolidationGroup)(nil)
)

//nolint:revive // valid struct name
type ConsolidationGroup struct {
	bun.BaseModel `bun:"table:consolidation_groups,alias:cg"`

	ID                  pulid.ID    `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID    `json:"businessUnitId"      bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID      pulid.ID    `json:"organizationId"      bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ConsolidationNumber string      `json:"consolidationNumber" bun:"consolidation_number,type:VARCHAR(100),notnull,unique"`
	Status              GroupStatus `json:"status"              bun:"status,type:consolidation_group_status_enum,notnull,default:'New'"`
	CancelReason        string      `json:"cancelReason"        bun:"cancel_reason,type:VARCHAR(100),nullzero"`
	CanceledByID        *pulid.ID   `json:"canceledById"        bun:"canceled_by_id,type:VARCHAR(100),nullzero"`
	CanceledAt          *int64      `json:"canceledAt"          bun:"canceled_at,type:BIGINT,nullzero"`

	// Metadata
	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	Shipments    []*shipment.Shipment       `json:"shipments,omitempty"    bun:"rel:has-many,join:id=consolidation_group_id"`
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	CanceledBy   *user.User                 `json:"canceledBy,omitempty"   bun:"rel:belongs-to,join:canceled_by_id=id"`
}

func (cg *ConsolidationGroup) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, cg,
		// validation.Field(&cg.Status, validation.Required.Error("Status is required")),
		validation.Field(&cg.Shipments,
			validation.Required.Error("Shipments are required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (cg *ConsolidationGroup) GetID() string {
	return cg.ID.String()
}

func (cg *ConsolidationGroup) GetTableName() string {
	return "consolidation_groups"
}

func (cg *ConsolidationGroup) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "cg",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "consolidation_number",
				Weight: "A",
				Type:   infra.PostgresSearchTypeComposite,
			},
			{
				Name:       "status",
				Weight:     "B",
				Type:       infra.PostgresSearchTypeEnum,
				Dictionary: "english",
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}

func (cg *ConsolidationGroup) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if cg.ID.IsNil() {
			cg.ID = pulid.MustNew("cg_")
		}

		cg.CreatedAt = now
	case *bun.UpdateQuery:
		cg.UpdatedAt = now
	}

	return nil
}
