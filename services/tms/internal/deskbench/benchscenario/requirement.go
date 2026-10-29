package benchscenario

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
)

type Requirement struct {
	Extension    agentextension.Type `yaml:"extension"    json:"extension,omitempty"`
	EmailProfile email.Purpose       `yaml:"emailProfile" json:"emailProfile,omitempty"`
	EDILading    string              `yaml:"ediLading"    json:"ediLading,omitempty"`
	DocumentText bool                `yaml:"documentText" json:"documentText,omitempty"`
}

var errRequirementShape = errors.New(
	"a requirement names one of extension, emailProfile, ediLading or documentText",
)

func (r *Requirement) validate() error {
	named := 0
	for _, set := range []bool{
		r.Extension != "", r.EmailProfile != "", r.EDILading != "", r.DocumentText,
	} {
		if set {
			named++
		}
	}
	switch {
	case named != 1:
		return errRequirementShape
	case r.Extension != "" && !r.Extension.IsValid():
		return fmt.Errorf("extension %q is not one of %v", r.Extension, agentextension.AllTypes())
	case r.EmailProfile != "" && !email.IsValidPurpose(r.EmailProfile):
		return fmt.Errorf("emailProfile %q is not an email purpose (%v)",
			r.EmailProfile, email.Purposes())
	case r.EDILading != "" && strings.TrimSpace(r.EDILading) == "":
		return errors.New("ediLading names the EDI partner by its code")
	default:
		return nil
	}
}

func (r *Requirement) Label() string {
	switch {
	case r.Extension != "":
		return "the " + string(r.Extension) + " extension enabled with an API key (AI Control → " +
			"Extensions)"
	case r.EmailProfile != "":
		return "an active email profile assigned to " + string(r.EmailProfile) + " or " +
			string(email.PurposeGeneral) + " email (email settings)"
	case r.DocumentText:
		return "the server reading an attached file's text (documentConversion configured " +
			"and document extraction available)"
	default:
		return "an active Lading version for EDI partner " + strings.TrimSpace(r.EDILading)
	}
}

func (r *Requirement) Probe() (query string, args []any) {
	switch {
	case r.Extension != "":
		return extensionReadySQL, []any{string(r.Extension), agentextension.ConfigKeyAPIKey}
	case r.DocumentText:
		return attachedTextReadySQL, nil
	case r.EmailProfile != "":
		return emailProfileReadySQL, []any{
			string(r.EmailProfile),
			string(email.PurposeGeneral),
			string(email.ProfileStatusActive),
		}
	default:
		return ladingReadySQL, []any{strings.TrimSpace(r.EDILading)}
	}
}

const (
	attachedTextReadySQL = `SELECT true`

	extensionReadySQL = `SELECT EXISTS (
		SELECT 1 FROM agent_extensions
		WHERE type = $1 AND enabled AND coalesce(configuration->>$2, '') <> '')`

	emailProfileReadySQL = `SELECT EXISTS (
		SELECT 1 FROM email_profile_assignments a
		JOIN email_profiles p ON p.id = a.profile_id
			AND p.organization_id = a.organization_id
			AND p.business_unit_id = a.business_unit_id
		WHERE a.purpose::text IN ($1, $2) AND p.status::text = $3)`

	ladingReadySQL = `SELECT EXISTS (
		SELECT 1 FROM edi_lading_projects lp
		JOIN edi_partners p ON p.id = lp.edi_partner_id
			AND p.organization_id = lp.organization_id
			AND p.business_unit_id = lp.business_unit_id
		WHERE lower(p.code) = lower($1) AND lp.active_version_id IS NOT NULL)`
)
