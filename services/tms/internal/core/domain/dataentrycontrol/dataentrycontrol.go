package dataentrycontrol

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*DataEntryControl)(nil)
	_ validationframework.TenantedEntity = (*DataEntryControl)(nil)
)

type DataEntryControl struct {
	bun.BaseModel `bun:"table:data_entry_controls,alias:dec" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`

	CodeCase  CaseFormat `json:"codeCase"  bun:"code_case,type:case_format_enum,notnull,default:'Upper'"`
	NameCase  CaseFormat `json:"nameCase"  bun:"name_case,type:case_format_enum,notnull,default:'TitleCase'"`
	EmailCase CaseFormat `json:"emailCase" bun:"email_case,type:case_format_enum,notnull,default:'Lower'"`
	CityCase  CaseFormat `json:"cityCase"  bun:"city_case,type:case_format_enum,notnull,default:'TitleCase'"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (dec *DataEntryControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(dec,
		validation.Field(&dec.CodeCase,
			validation.Required.Error("Code case is required"),
			validation.By(func(value any) error {
				c, ok := value.(CaseFormat)
				if !ok {
					return errors.New("invalid case format type")
				}
				if !c.IsValid() {
					return errors.New(
						"code case must be one of: AsEntered, Upper, Lower, TitleCase",
					)
				}
				return nil
			}),
		),
		validation.Field(&dec.NameCase,
			validation.Required.Error("Name case is required"),
			validation.By(func(value any) error {
				c, ok := value.(CaseFormat)
				if !ok {
					return errors.New("invalid case format type")
				}
				if !c.IsValid() {
					return errors.New(
						"name case must be one of: AsEntered, Upper, Lower, TitleCase",
					)
				}
				return nil
			}),
		),
		validation.Field(&dec.EmailCase,
			validation.Required.Error("Email case is required"),
			validation.By(func(value any) error {
				c, ok := value.(CaseFormat)
				if !ok {
					return errors.New("invalid case format type")
				}
				if !c.IsValid() {
					return errors.New(
						"email case must be one of: AsEntered, Upper, Lower, TitleCase",
					)
				}
				return nil
			}),
		),
		validation.Field(&dec.CityCase,
			validation.Required.Error("City case is required"),
			validation.By(func(value any) error {
				c, ok := value.(CaseFormat)
				if !ok {
					return errors.New("invalid case format type")
				}
				if !c.IsValid() {
					return errors.New(
						"city case must be one of: AsEntered, Upper, Lower, TitleCase",
					)
				}
				return nil
			}),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (dec *DataEntryControl) GetTableName() string {
	return "data_entry_controls"
}

func (dec *DataEntryControl) GetID() pulid.ID {
	return dec.ID
}

func (dec *DataEntryControl) GetOrganizationID() pulid.ID {
	return dec.OrganizationID
}

func (dec *DataEntryControl) GetBusinessUnitID() pulid.ID {
	return dec.BusinessUnitID
}

func (dec *DataEntryControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dec.ID.IsNil() {
			dec.ID = pulid.MustNew("dec_")
		}
		dec.CreatedAt = now
	case *bun.UpdateQuery:
		dec.UpdatedAt = now
	}

	return nil
}

func NewDefaultDataEntryControl(orgID, buID pulid.ID) *DataEntryControl {
	return &DataEntryControl{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CodeCase:       CaseFormatUpper,
		NameCase:       CaseFormatTitleCase,
		EmailCase:      CaseFormatLower,
		CityCase:       CaseFormatTitleCase,
	}
}
