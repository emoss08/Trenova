package supportaccess

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

// This file is the whole rule for what a Trenova support session may do. A read-only grant
// reads; a full grant has an organization administrator's reach except what is listed here,
// which no grant ever reaches. The permission engine applies it to the support principal and
// nothing else, so a resource missing from the registry, or an operation a resource does not
// declare, is refused as it is for everyone.
//
// A resource of a sensitive kind (SensitiveResource) must appear in denied or in reviewed:
// TestEverySensitiveResourceIsClassifiedForSupport fails until a new one is, so adding a
// resource that holds a secret or controls sign-in forces someone to decide what Trenova staff
// may do with it.

type denial int

const (
	// denyAll refuses every operation, reads included: the resource holds a secret or a way in.
	denyAll denial = iota + 1
	// denyWrites refuses every operation but read.
	denyWrites
)

type rule struct {
	denial denial
	reason string
}

var denied = map[permission.Resource]rule{ //nolint:exhaustive // a deny list: a resource not named is allowed by scope
	permission.ResourceUser:                  {denyWrites, "managing users"},
	permission.ResourceRole:                  {denyWrites, "managing roles"},
	permission.ResourceAccessPolicy:          {denyWrites, "sign-in settings"},
	permission.ResourceMFAAuthenticator:      {denyWrites, "sign-in settings"},
	permission.ResourceExternalIdentity:      {denyWrites, "sign-in settings"},
	permission.ResourceIdentityProvider:      {denyAll, "single sign-on configuration holds a client secret"},
	permission.ResourceSCIMDirectory:         {denyAll, "provisioning tokens and the roles they grant"},
	permission.ResourceAPIKey:                {denyAll, "creating API keys"},
	permission.ResourceSupportAccess:         {denyAll, "support access grants are the customer's alone"},
	permission.ResourceIntegration:           {denyAll, "integration credentials"},
	permission.ResourceAccountingIntegration: {denyAll, "accounting connection credentials"},
	permission.ResourceEmailProfile:          {denyAll, "mail server and provider credentials"},
	permission.ResourceInboundMailbox:        {denyAll, "mailbox credentials"},
	permission.ResourceAIProvider:            {denyAll, "AI provider keys"},
	permission.ResourcePortalAutomation:      {denyAll, "stored carrier portal sign-ins"},
	permission.ResourceProtectedPersonal:     {denyAll, "drivers' personal details are not needed to support the software"},
	permission.ResourceCaptureDevice:         {denyWrites, "pairing a device is a lasting way in"},
	permission.ResourceDriverPortal:          {denyWrites, "inviting a driver gives a sign-in"},
	permission.ResourceSecurityAuditLog:      {denyWrites, "security alerts and reviews are the customer's"},
	permission.ResourceAuditLegalHold:        {denyWrites, "legal holds are the customer's decision"},
	permission.ResourcePlatformCatalog:       {denyWrites, "what the plan includes"},
	permission.ResourceAgentControl:          {denyWrites, "money limits for agents"},
	permission.ResourceAgentDefinition:       {denyWrites, "an agent keeps acting after the grant ends"},
	permission.ResourceAgentExtension:        {denyWrites, "an extension keeps acting after the grant ends"},
	permission.ResourceFactoringCompany:      {denyWrites, "factoring remittance details"},
	permission.ResourceFactoringAgreement:    {denyWrites, "factoring terms and where funds go"},
	permission.ResourceFactoringRemittance:   {denyWrites, "factoring remittance details"},
	permission.ResourceDocumentTemplate:      {denyWrites, "invoice templates carry billing bank details"},
	permission.ResourceEDI:                   {denyWrites, "trading partner keys and endpoints"},
}

// reviewed lists the sensitive resources a support session may use as its scope allows, each
// with why that is safe.
var reviewed = map[permission.Resource]string{ //nolint:exhaustive // an allow list of reviewed sensitive resources
	permission.ResourceOrganization:          "organization settings hold no secret; sign-in settings have their own resources",
	permission.ResourceBusinessUnit:          "business unit details",
	permission.ResourceTableConfiguration:    "saved table views",
	permission.ResourceCustomFieldDefinition: "custom field definitions",
	permission.ResourceSequenceConfig:        "number sequences",
	permission.ResourceEmailLog:              "what was sent, never a credential",
	permission.ResourceEmailSuppression:      "suppressed addresses",
	permission.ResourceTableChangeAlert:      "table change alerts",
	permission.ResourceHomeLayoutPreset:      "home screen presets",
	permission.ResourceDataEntryControl:      "data entry rules",
	permission.ResourceAuditLog:              "reading the audit log; exports are refused to every support session",
	permission.ResourceAuthEvent:             "sign-in activity, read-only",
	permission.ResourceRiskDecision:          "sign-in risk decisions, read-only",
	permission.ResourceProvisioningAudit:     "provisioning activity, read-only",
	permission.ResourceAIAuditTrail:          "agent run records, read-only",
	permission.ResourceAgentRun:              "agent runs",
	permission.ResourceAgentException:        "agent exceptions",
	permission.ResourceAgentEvalSuite:        "agent evaluation suites",
	permission.ResourceAgentFeedback:         "agent feedback",
	permission.ResourceWorkerCredential:      "licences and certificates a driver holds, not sign-in credentials",
	permission.ResourceWorkerCredentialType:  "kinds of driver credential",
	permission.ResourceFactoringNoa:          "notices of assignment",
	permission.ResourceFactoringSubmission:   "factoring submissions",
	permission.ResourceFactoringReserve:      "factoring reserves",
	permission.ResourceProtectedMargins:      "margins, which support reads to explain a rate",
}

// SensitiveCategories are the role editor categories whose every resource must be classified.
var SensitiveCategories = []string{"Users & security", "Organization settings", "AI & agents"}

// sensitiveNameParts catch a resource of a secret-holding kind filed under another category.
var sensitiveNameParts = []string{
	"integration", "mailbox", "provider", "api_key", "credential", "scim", "factoring",
	"remittance", "portal", "device", "protected_", "support",
}

// SensitiveResource reports whether a resource must be named in the support policy.
func SensitiveResource(resource, category string) bool {
	for _, c := range SensitiveCategories {
		if c == category {
			return true
		}
	}
	for _, part := range sensitiveNameParts {
		if strings.Contains(resource, part) {
			return true
		}
	}
	return false
}

// Classified reports whether the policy names the resource, as denied or reviewed.
func Classified(resource string) bool {
	r := permission.Resource(resource)
	if _, ok := denied[r]; ok {
		return true
	}
	_, ok := reviewed[r]
	return ok
}

// DeniedReason is why no support session may use the resource as op, or "" when the policy
// does not refuse it.
func DeniedReason(resource string, op permission.Operation) string {
	if op == permission.OpExport {
		return "exporting data in bulk"
	}
	r, ok := denied[permission.Resource(resource)]
	if !ok {
		return ""
	}
	if r.denial == denyAll || op != permission.OpRead {
		return r.reason
	}
	return ""
}

// Allows is the rule on the support principal: whether a session under a grant of scope may
// perform op on resource. Every non-read operation is refused to a read-only grant, and the
// deny list is refused to every grant.
func Allows(scope Scope, resource string, op permission.Operation) bool {
	if !scope.IsValid() {
		return false
	}
	if DeniedReason(resource, op) != "" {
		return false
	}
	if scope == ScopeReadOnly {
		return op == permission.OpRead
	}
	return true
}

// Effective is what a grant of scope holds over a whole registry: every declared operation the
// policy allows, less any operation a parent would carry to a child the policy refuses it on.
// The permission engine lets a grant on a parent satisfy a check on its child, so a refused
// child operation is taken off its parent too: support may update factoring reserves only where
// that cannot also update a factoring remittance.
func Effective(scope Scope, defs []*permissionspec.ResourceDefinition) map[string][]permission.Operation {
	out := make(map[string][]permission.Operation, len(defs))
	for _, def := range defs {
		for _, op := range def.Operations {
			if Allows(scope, def.Resource, op.Operation) {
				out[def.Resource] = append(out[def.Resource], op.Operation)
			}
		}
	}
	for _, def := range defs {
		if def.ParentResource == "" {
			continue
		}
		for _, op := range def.Operations {
			if Allows(scope, def.Resource, op.Operation) {
				continue
			}
			out[def.ParentResource] = slices.DeleteFunc(out[def.ParentResource], func(held permission.Operation) bool {
				return held == op.Operation
			})
		}
	}
	for resource, ops := range out {
		if len(ops) == 0 {
			delete(out, resource)
		}
	}
	return out
}
