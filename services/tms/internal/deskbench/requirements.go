package deskbench

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
)

type Requirement struct {
	Extension    agentextension.Type `yaml:"extension"    json:"extension,omitempty"`
	EmailProfile email.Purpose       `yaml:"emailProfile" json:"emailProfile,omitempty"`
	EDILading    string              `yaml:"ediLading"    json:"ediLading,omitempty"`
}

var errRequirementShape = errors.New(
	"a requirement names one of extension, emailProfile or ediLading",
)

func (r *Requirement) validate() error {
	named := 0
	for _, set := range []bool{r.Extension != "", r.EmailProfile != "", r.EDILading != ""} {
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
	default:
		return "an active Lading version for EDI partner " + strings.TrimSpace(r.EDILading)
	}
}

func (r *Requirement) probe() (string, []any) {
	switch {
	case r.Extension != "":
		return extensionReadySQL, []any{string(r.Extension), agentextension.ConfigKeyAPIKey}
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

func (b *Bench) unmetRequirements(
	ctx context.Context,
	session *Session,
	requires []Requirement,
) (string, error) {
	if len(requires) == 0 {
		return "", nil
	}

	unmet, err := dbtx.Read(session.Context(ctx), b.DB, func(ctx context.Context) ([]string, error) {
		db := b.DB.Keel(ctx)
		unmet := make([]string, 0, len(requires))
		for idx := range requires {
			statement, args := requires[idx].probe()
			ready, err := keel.QueryValue[bool](ctx, db, statement, args...)
			if err != nil {
				return nil, fmt.Errorf("check the scenario needs %s: %w", requires[idx].Label(), err)
			}
			if !ready {
				unmet = append(unmet, requires[idx].Label())
			}
		}

		return unmet, nil
	})
	if err != nil || len(unmet) == 0 {
		return "", err
	}

	return "needs " + strings.Join(unmet, "; and "), nil
}
