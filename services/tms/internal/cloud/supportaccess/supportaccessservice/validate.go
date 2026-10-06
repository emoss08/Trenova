package supportaccessservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	principalNamePrefix  = "Trenova Support ("
	principalNameSuffix  = ")"
	principalUserPrefix  = "support-"
	principalSuffixChars = 10
	principalPasswordLen = 32
)

func validateStart(req *StartSessionRequest) error {
	multiErr := errortypes.NewMultiError()
	multiErr.AddOzzoError(validation.ValidateStruct(
		req,
		validation.Field(
			&req.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(
			&req.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&req.Reason, supportaccess.ReasonRules("Reason")...),
		validation.Field(
			&req.TicketReference,
			validation.RuneLength(0, supportaccess.MaxTicketLength).
				Error("Ticket reference must be at most 100 characters"),
		),
	))
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateElevation(req *ElevateRequest) error {
	multiErr := errortypes.NewMultiError()
	multiErr.AddOzzoError(validation.ValidateStruct(
		req,
		validation.Field(&req.Password, validation.Required.Error("Password is required")),
		validation.Field(
			&req.Code,
			validation.When(
				strings.TrimSpace(req.RecoveryCode) == "",
				validation.Required.Error("Enter the code from your authenticator app"),
			),
		),
		validation.Field(&req.Reason, supportaccess.ReasonRules("Reason")...),
		validation.Field(
			&req.TicketReference,
			validation.Required.Error("A ticket reference is required to change customer data"),
			validation.RuneLength(1, supportaccess.MaxTicketLength).
				Error("Ticket reference must be at most 100 characters"),
		),
	))
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func PrincipalName(staffName string) string {
	name := strings.TrimSpace(staffName)
	if name == "" {
		name = "staff"
	}
	limit := maxNameLength - len([]rune(principalNamePrefix)) - len([]rune(principalNameSuffix))
	return principalNamePrefix + truncate(name, limit) + principalNameSuffix
}

type principalUserParams struct {
	Staff          *tenant.User
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	EmailDomain    string
	Now            int64
}

func newPrincipalUser(p *principalUserParams) (*tenant.User, error) {
	id := pulid.MustNew(tenant.UserIDPrefix)
	raw := strings.ToLower(id.String())
	suffix := raw[len(raw)-principalSuffixChars:]

	secret, err := tokenutils.RandomHex(principalPasswordLen)
	if err != nil {
		return nil, err
	}

	usr := &tenant.User{
		ID:                    id,
		BusinessUnitID:        p.BusinessUnitID,
		CurrentOrganizationID: p.OrganizationID,
		Status:                domaintypes.StatusInactive,
		Name:                  PrincipalName(p.Staff.Name),
		Username:              principalUserPrefix + suffix,
		TimeFormat:            p.Staff.TimeFormat,
		EmailAddress:          principalUserPrefix + raw + "@" + p.EmailDomain,
		Timezone:              p.Staff.Timezone,
		Locale:                p.Staff.Locale,
		CreatedAt:             p.Now,
		UpdatedAt:             p.Now,
	}
	if usr.TimeFormat == "" {
		usr.TimeFormat = domaintypes.TimeFormat12Hour
	}
	if usr.Timezone == "" {
		usr.Timezone = "UTC"
	}
	if usr.Locale == "" {
		usr.Locale = "en"
	}

	if usr.Password, err = usr.GeneratePassword(secret); err != nil {
		return nil, err
	}

	return usr, nil
}
