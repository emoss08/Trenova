package ratetable

import (
	"context"
	"errors"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type LookupType string

const (
	LookupTypeExact = LookupType("Exact")
	LookupTypeRange = LookupType("Range")
)

func (lt LookupType) String() string {
	return string(lt)
}

func LookupTypeFromString(s string) (LookupType, error) {
	switch s {
	case "Exact":
		return LookupTypeExact, nil
	case "Range":
		return LookupTypeRange, nil
	default:
		return "", errors.New("invalid lookup type")
	}
}

var keyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

var (
	_ bun.BeforeAppendModelHook          = (*RateTable)(nil)
	_ validationframework.TenantedEntity = (*RateTable)(nil)
	_ domaintypes.PostgresSearchable     = (*RateTable)(nil)
)

type RateTable struct {
	bun.BaseModel             `bun:"table:rate_tables,alias:rtb" json:"-"`
	pagination.CursorValueSet `bun:",embed"                      json:"-"`

	ID             pulid.ID   `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID   `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID   `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Name           string     `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Key            string     `json:"key"            bun:"key,type:VARCHAR(64),notnull"`
	Description    string     `json:"description"    bun:"description,type:TEXT,nullzero"`
	LookupType     LookupType `json:"lookupType"     bun:"lookup_type,type:rate_table_lookup_type_enum,notnull"`
	Active         bool       `json:"active"         bun:"active,type:BOOLEAN,notnull,default:true"`
	Version        int64      `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64      `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64      `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string     `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string     `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-"                 bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"                 bun:"rel:belongs-to,join:organization_id=id"`
	Entries      []*RateTableEntry    `json:"entries,omitempty" bun:"rel:has-many,join:id=rate_table_id"`
}

func (rt *RateTable) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(rt,
		validation.Field(&rt.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&rt.Key,
			validation.Required,
			validation.Length(1, 64),
			validation.Match(keyPattern).
				Error("Key must start with a letter and contain only letters, digits, and underscores"),
		),
		validation.Field(&rt.LookupType, validation.Required, validation.In(
			LookupTypeExact,
			LookupTypeRange,
		)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	rt.validateEntries(multiErr)
}

func (rt *RateTable) validateEntries(multiErr *errortypes.MultiError) {
	switch rt.LookupType {
	case LookupTypeExact:
		validateExactEntries(rt.Entries, multiErr)
	case LookupTypeRange:
		validateRangeEntries(rt.Entries, multiErr)
	}
}

func (rt *RateTable) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rt.ID.IsNil() {
			rt.ID = pulid.MustNew("rt_")
		}
		rt.CreatedAt = now
		rt.UpdatedAt = now
	case *bun.UpdateQuery:
		rt.UpdatedAt = now
	}

	return nil
}

func (rt *RateTable) GetID() pulid.ID {
	return rt.ID
}

func (rt *RateTable) GetCreatedAt() int64 {
	return rt.CreatedAt
}

func (rt *RateTable) GetOrganizationID() pulid.ID {
	return rt.OrganizationID
}

func (rt *RateTable) GetBusinessUnitID() pulid.ID {
	return rt.BusinessUnitID
}

func (rt *RateTable) GetTableName() string {
	return "rate_tables"
}

func (rt *RateTable) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rtb",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "key", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}
