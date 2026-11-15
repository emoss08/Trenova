package tenant

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	PlaceholderSCAC       = "TBDX"
	PlaceholderDOTNumber  = "0"
	PlaceholderCity       = "Pending"
	PlaceholderPostalCode = "00000"
)

const (
	SetupFieldName         = "name"
	SetupFieldTimezone     = "timezone"
	SetupFieldAddressLine1 = "addressLine1"
	SetupFieldCity         = "city"
	SetupFieldStateID      = "stateId"
	SetupFieldPostalCode   = "postalCode"
	SetupFieldSCACCode     = "scacCode"
	SetupFieldDOTNumber    = "dotNumber"
)

type OrganizationSetup struct {
	Complete    bool     `json:"complete"`
	Missing     []string `json:"missing"`
	CanComplete bool     `json:"canComplete"`
}

func (o *Organization) SetupMissing() []string {
	missing := make([]string, 0, 8)
	add := func(field string, unset bool) {
		if unset {
			missing = append(missing, field)
		}
	}

	add(SetupFieldName, blank(o.Name))
	add(SetupFieldTimezone, blank(o.Timezone))
	add(SetupFieldAddressLine1, blank(o.AddressLine1))
	add(SetupFieldCity, blank(o.City) || o.City == PlaceholderCity)
	add(SetupFieldStateID, o.StateID.IsNil())
	add(SetupFieldPostalCode, blank(o.PostalCode) || o.PostalCode == PlaceholderPostalCode)
	add(SetupFieldSCACCode, blank(o.ScacCode) || isPlaceholderSCAC(o.ScacCode))
	add(SetupFieldDOTNumber, blank(o.DOTNumber) || o.DOTNumber == PlaceholderDOTNumber)

	return missing
}

func (o *Organization) Setup() *OrganizationSetup {
	missing := o.SetupMissing()
	return &OrganizationSetup{Complete: len(missing) == 0, Missing: missing}
}

func blank(value string) bool {
	return strings.TrimSpace(value) == ""
}

func isPlaceholderSCAC(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), PlaceholderSCAC)
}

func refusePlaceholderSCAC(value any) error {
	scac, _ := value.(string)
	if isPlaceholderSCAC(scac) {
		return validation.NewError("validation_placeholder_scac", "Enter your company's SCAC code")
	}
	return nil
}
