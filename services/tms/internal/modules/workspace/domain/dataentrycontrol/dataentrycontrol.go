package dataentrycontrol

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*DataEntryControl)(nil)
)

type DataEntryControl struct {
	keel.Table `                                                   keel:"data_entry_controls"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:dec_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	CodeCase  CaseFormat `json:"codeCase" keel:"code_case"`
	NameCase  CaseFormat `json:"nameCase" keel:"name_case"`
	EmailCase CaseFormat `json:"emailCase" keel:"email_case"`
	CityCase  CaseFormat `json:"cityCase" keel:"city_case"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (dec *DataEntryControl) BeforeCreate(_ context.Context) error {
	if dec.CodeCase == "" {
		dec.CodeCase = CaseFormatUpper
	}
	if dec.NameCase == "" {
		dec.NameCase = CaseFormatTitleCase
	}
	if dec.EmailCase == "" {
		dec.EmailCase = CaseFormatLower
	}
	if dec.CityCase == "" {
		dec.CityCase = CaseFormatTitleCase
	}

	return nil
}

func (dec *DataEntryControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(dec,
		validation.Field(
			&dec.CodeCase,
			validation.Required.Error("Code case is required"),
			domainvalidation.ValidEnum[CaseFormat](
				"code case must be one of: AsEntered, Upper, Lower, TitleCase",
			),
		),
		validation.Field(
			&dec.NameCase,
			validation.Required.Error("Name case is required"),
			domainvalidation.ValidEnum[CaseFormat](
				"name case must be one of: AsEntered, Upper, Lower, TitleCase",
			),
		),
		validation.Field(
			&dec.EmailCase,
			validation.Required.Error("Email case is required"),
			domainvalidation.ValidEnum[CaseFormat](
				"email case must be one of: AsEntered, Upper, Lower, TitleCase",
			),
		),
		validation.Field(
			&dec.CityCase,
			validation.Required.Error("City case is required"),
			domainvalidation.ValidEnum[CaseFormat](
				"city case must be one of: AsEntered, Upper, Lower, TitleCase",
			),
		),
	))
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
