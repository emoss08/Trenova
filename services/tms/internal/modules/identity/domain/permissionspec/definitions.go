package permissionspec

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type BundleGrant struct {
	Resource   permission.Resource    `json:"resource"`
	Operations []permission.Operation `json:"operations"`
	Scope      permission.DataScope   `json:"scope,omitempty"`
}

// ShareRecords lets a person share what they can read. It rides with the
// bundles that put people where records are shared from: the Desk, and the
// billing work whose invoices and disputes go to someone else to look at.
func ShareRecords() BundleGrant {
	return BundleGrant{
		Resource:   permission.ResourceRecordShare,
		Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
	}
}

type Bundle struct {
	ID          string        `json:"id"`
	Module      string        `json:"module"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Grants      []BundleGrant `json:"grants"`
}

type JobTemplate struct {
	ID             string                      `json:"id"`
	Label          string                      `json:"label"`
	Description    string                      `json:"description"`
	Bundles        []string                    `json:"bundles"`
	MaxSensitivity permission.FieldSensitivity `json:"maxSensitivity"`
	ReadEverything bool                        `json:"readEverything,omitempty"`
	ReadExcept     []permission.Resource       `json:"readExcept,omitempty"`
}

type OperationDefinition struct {
	Operation   permission.Operation   `json:"operation"`
	DisplayName string                 `json:"displayName"`
	Description string                 `json:"description"`
	Implies     []permission.Operation `json:"implies"`
}

type ResourceDefinition struct {
	Resource           string                                 `json:"resource"`
	DisplayName        string                                 `json:"displayName"`
	Description        string                                 `json:"description"`
	Category           string                                 `json:"category"`
	Operations         []OperationDefinition                  `json:"operations"`
	CompositeOps       map[string][]permission.Operation      `json:"compositeOps,omitempty"`
	ParentResource     string                                 `json:"parentResource,omitempty"`
	FieldSensitivities map[string]permission.FieldSensitivity `json:"fieldSensitivities,omitempty"`
	DefaultSensitivity permission.FieldSensitivity            `json:"defaultSensitivity"`
	Scopes             []ScopeSupport                         `json:"scopes,omitempty"`
	ReferenceView      []string                               `json:"referenceView,omitempty"`
}

type ScopeEnforcement string

const (
	ScopeEnforcedByApplication ScopeEnforcement = "application"
	ScopeEnforcedByDatabase    ScopeEnforcement = "database"
)

type ScopeSupport struct {
	Scope      permission.DataScope `json:"scope"`
	EnforcedBy ScopeEnforcement     `json:"enforcedBy"`
}
