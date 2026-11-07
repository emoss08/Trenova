package audit

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type AlertKind string

const (
	AlertRoleChanged           = AlertKind("RoleChanged")
	AlertAPIKeyChanged         = AlertKind("APIKeyChanged")
	AlertMFAReset              = AlertKind("MFAReset")
	AlertIdentityConfigChanged = AlertKind("IdentityConfigChanged")
	AlertCrossTenantRefused    = AlertKind("CrossTenantRefused")
	AlertMalwareRefused        = AlertKind("MalwareRefused")
	AlertLegalHoldChanged      = AlertKind("LegalHoldChanged")
	AlertBulkDelete            = AlertKind("BulkDelete")
	AlertBulkExport            = AlertKind("BulkExport")
)

type AlertDefinition struct {
	Kind          AlertKind
	Title         string
	Description   string
	Threshold     int
	WindowMinutes int
	Counted       bool
}

var AlertDefinitions = []AlertDefinition{
	{
		Kind:        AlertRoleChanged,
		Title:       "Role or role assignment changed",
		Description: "A role was created, changed or assigned, or someone's roles were removed",
		Threshold:   1, WindowMinutes: 1,
	},
	{
		Kind:        AlertAPIKeyChanged,
		Title:       "API key created, rotated or revoked",
		Description: "Someone created, rotated, changed or revoked an API key",
		Threshold:   1, WindowMinutes: 1,
	},
	{
		Kind:          AlertMFAReset,
		Title:         "Two-factor turned off or reset",
		Description:   "An authenticator was turned off, or an administrator reset someone's two-factor",
		Threshold:     1,
		WindowMinutes: 1,
	},
	{
		Kind:        AlertIdentityConfigChanged,
		Title:       "Sign-in configuration changed",
		Description: "An identity provider, SCIM directory or access policy was changed",
		Threshold:   1, WindowMinutes: 1,
	},
	{
		Kind:        AlertCrossTenantRefused,
		Title:       "Request for another organization refused",
		Description: "A request reached for another organization's records and was refused",
		Threshold:   1, WindowMinutes: 1,
	},
	{
		Kind:        AlertMalwareRefused,
		Title:       "File refused by the malware scanner",
		Description: "A file sent to Trenova was refused as malware",
		Threshold:   1, WindowMinutes: 1,
	},
	{
		Kind:        AlertLegalHoldChanged,
		Title:       "Legal hold placed or released",
		Description: "Someone placed or released a legal hold on the audit trail",
		Threshold:   1, WindowMinutes: 1,
	},
	{
		Kind:        AlertBulkDelete,
		Title:       "Many records deleted",
		Description: "One person, key or agent deleted many records in a short time",
		Threshold:   50, WindowMinutes: 10, Counted: true,
	},
	{
		Kind:        AlertBulkExport,
		Title:       "Many exports or downloads",
		Description: "One person, key or agent exported or downloaded many times in a short time",
		Threshold:   25, WindowMinutes: 10, Counted: true,
	},
}

func AlertDefinitionFor(kind AlertKind) (AlertDefinition, bool) {
	for _, def := range AlertDefinitions {
		if def.Kind == kind {
			return def, true
		}
	}
	return AlertDefinition{}, false
}

//nolint:exhaustive // only resources whose changes raise an alert are listed
var securityAlertResources = map[permission.Resource]AlertKind{
	permission.ResourceRole:             AlertRoleChanged,
	permission.ResourceAPIKey:           AlertAPIKeyChanged,
	permission.ResourceMFAAuthenticator: AlertMFAReset,
	permission.ResourceIdentityProvider: AlertIdentityConfigChanged,
	permission.ResourceSCIMDirectory:    AlertIdentityConfigChanged,
	permission.ResourceAccessPolicy:     AlertIdentityConfigChanged,
}

func AlertKindOf(e *Entry) (AlertKind, bool) {
	if e == nil {
		return "", false
	}
	if e.Category != CategorySecurity {
		return "", false
	}
	if e.Action == ActionLegalHoldPlaced || e.Action == ActionLegalHoldReleased {
		return AlertLegalHoldChanged, true
	}
	if e.Action == ActionMalwareRefused {
		return AlertMalwareRefused, true
	}
	if e.Action == ActionCrossTenantRefused {
		return AlertCrossTenantRefused, true
	}
	if e.Resource == permission.ResourceMFAAuthenticator &&
		e.Operation != permission.OpDelete && e.Operation != permission.OpUpdate {
		return "", false
	}
	kind, ok := securityAlertResources[e.Resource]
	return kind, ok
}

func CountedAlertKindOf(e *Entry) (AlertKind, bool) {
	if e == nil {
		return "", false
	}
	switch e.Operation { //nolint:exhaustive // only deletes and exports are counted
	case permission.OpDelete:
		return AlertBulkDelete, true
	case permission.OpExport:
		return AlertBulkExport, true
	case permission.OpRead:
		if access, _ := e.Metadata["access"].(string); access == "download" {
			return AlertBulkExport, true
		}
	}
	return "", false
}

type AlertPolicy struct {
	keel.Table `keel:"audit_alert_policies"`

	ID             pulid.ID  `json:"id"             keel:"id,pk,autoid:aap_"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID  `json:"organizationId" keel:"organization_id,pk,tenant"`
	Kind           AlertKind `json:"kind"           keel:"kind"`
	Enabled        bool      `json:"enabled"        keel:"enabled"`
	Threshold      int       `json:"threshold"      keel:"threshold"`
	WindowMinutes  int       `json:"windowMinutes"  keel:"window_minutes"`
	Version        int64     `json:"version"        keel:"version,version"`
	CreatedAt      int64     `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64     `json:"updatedAt"      keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (p *AlertPolicy) GetTableName() string {
	return "audit_alert_policies"
}

func DefaultAlertPolicy(def AlertDefinition) *AlertPolicy {
	return &AlertPolicy{
		Kind:          def.Kind,
		Enabled:       true,
		Threshold:     def.Threshold,
		WindowMinutes: def.WindowMinutes,
	}
}

func (p *AlertPolicy) Validate(multiErr *errortypes.MultiError) {
	def, ok := AlertDefinitionFor(p.Kind)
	if !ok {
		multiErr.Add("kind", errortypes.ErrInvalid, "Trenova raises no such alert")
		return
	}
	if !def.Counted {
		p.Threshold = 1
		p.WindowMinutes = 1
		return
	}
	if p.Threshold < 2 || p.Threshold > 100000 {
		multiErr.Add(
			"threshold",
			errortypes.ErrInvalid,
			"Alert after between 2 and 100,000 entries",
		)
	}
	if p.WindowMinutes < 1 || p.WindowMinutes > 1440 {
		multiErr.Add(
			"windowMinutes",
			errortypes.ErrInvalid,
			"Count over between 1 minute and 24 hours",
		)
	}
}
