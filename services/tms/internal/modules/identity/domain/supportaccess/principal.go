package supportaccess

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// PrincipalIDPrefix marks the users row that stands for Trenova support under one grant. Every
// foreign key and audit entry that needs a user id names it, so nothing done under a grant is
// credited to one of the organization's people. The prefix is how the permission engine routes
// a check to the grant instead of to role assignments, and how people lists leave it out; the
// engine still reads the live grant, so a prefix alone grants nothing.
const PrincipalIDPrefix = "sup_"

const (
	principalEmailDomain = "support.trenova.invalid"
	principalTimezone    = "UTC"
	principalNamePrefix  = "Trenova support: "
	principalUserPrefix  = "sup-"
	principalUserRandom  = 16
)

func IsPrincipalID(id pulid.ID) bool {
	return strings.HasPrefix(id.String(), PrincipalIDPrefix)
}

// PrincipalName is the support principal's name as people read it in the audit log.
func PrincipalName(staffEmail string) string {
	return principalNamePrefix + staffEmail
}

type PrincipalUserParams struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	StaffEmail     string
	Password       string
}

// NewPrincipalUser builds the inactive, system-owned users row a grant is attributed to. It
// never signs in: it is inactive, its password is random and unknown, its address is under a
// reserved name that can receive no mail, and authservice refuses to issue it a sign-in
// session whatever its state.
func NewPrincipalUser(p PrincipalUserParams) *tenant.User {
	id := pulid.MustNew(PrincipalIDPrefix)
	suffix := strings.ToLower(id.String())
	suffix = suffix[len(suffix)-principalUserRandom:]
	return &tenant.User{
		ID:                    id,
		BusinessUnitID:        p.BusinessUnitID,
		CurrentOrganizationID: p.OrganizationID,
		Status:                domaintypes.StatusInactive,
		Name:                  PrincipalName(p.StaffEmail),
		Username:              principalUserPrefix + suffix,
		EmailAddress:          principalUserPrefix + suffix + "@" + principalEmailDomain,
		Password:              p.Password,
		Timezone:              principalTimezone,
		AssistantHidden:       true,
	}
}
