package supportaccess

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// This file is the whole rule for what a Trenova support session may do. A read-only grant
// reads; a full grant has an organization administrator's reach except what is listed here,
// which no grant ever reaches. The permission engine applies it to the support principal and
// nothing else, so a resource missing from the registry, or an operation a resource does not
// declare, is refused as it is for everyone. Where money is sent is guarded column by column in
// the database (paymentDestinations), because the resources that hold it are ones support must
// still be able to change.
//
// A resource of a sensitive kind (SensitiveResource) must appear in denied or in reviewed:
// TestEverySensitiveResourceIsClassifiedForSupport fails until a new one is, so adding a
// resource that holds a secret or controls sign-in forces someone to decide what Trenova staff
// may do with it.

type denial int

const (
	denyAll denial = iota + 1
	denyWrites
)

type rule struct {
	denial denial
	limit  Limit
}

var denied = map[permission.Resource]rule{ //nolint:exhaustive // a deny list: a resource not named is allowed by scope
	permission.ResourceUser:                  {denyWrites, LimitPeople},
	permission.ResourceRole:                  {denyWrites, LimitPeople},
	permission.ResourceAccessPolicy:          {denyWrites, LimitPeople},
	permission.ResourceMFAAuthenticator:      {denyWrites, LimitPeople},
	permission.ResourceExternalIdentity:      {denyWrites, LimitPeople},
	permission.ResourceIdentityProvider:      {denyAll, LimitSingleSignOn},
	permission.ResourceSCIMDirectory:         {denyAll, LimitSingleSignOn},
	permission.ResourceAPIKey:                {denyAll, LimitAPIKeys},
	permission.ResourceSupportAccess:         {denyAll, LimitSupportAccess},
	permission.ResourceIntegration:           {denyAll, LimitCredentials},
	permission.ResourceAccountingIntegration: {denyAll, LimitCredentials},
	permission.ResourceEmailProfile:          {denyAll, LimitCredentials},
	permission.ResourceInboundMailbox:        {denyAll, LimitCredentials},
	permission.ResourceAIProvider:            {denyAll, LimitCredentials},
	permission.ResourcePortalAutomation:      {denyAll, LimitCredentials},
	permission.ResourceProtectedPersonal:     {denyAll, LimitDriverPersonal},
	permission.ResourceCaptureDevice:         {denyWrites, LimitDevices},
	permission.ResourceDriverPortal:          {denyWrites, LimitDevices},
	permission.ResourceSecurityAuditLog:      {denyWrites, LimitSecurityReview},
	permission.ResourceAuditLegalHold:        {denyWrites, LimitSecurityReview},
	permission.ResourcePlatformCatalog:       {denyWrites, LimitPlan},
	permission.ResourceAgentControl:          {denyWrites, LimitAgents},
	permission.ResourceAgentDefinition:       {denyWrites, LimitAgents},
	permission.ResourceAgentExtension:        {denyWrites, LimitAgents},
	permission.ResourceFactoringCompany:      {denyWrites, LimitFactoring},
	permission.ResourceFactoringAgreement:    {denyWrites, LimitFactoring},
	permission.ResourceFactoringRemittance:   {denyWrites, LimitFactoring},
	permission.ResourceDocumentTemplate:      {denyWrites, LimitDocumentTemplates},
	permission.ResourceEDI:                   {denyWrites, LimitEDI},
	permission.ResourceCustomerCommunication: {denyWrites, LimitOutbound},
}

// reviewed lists the sensitive resources a support session may use as its scope allows, each
// with why that is safe.
var reviewed = map[permission.Resource]string{ //nolint:exhaustive // an allow list of reviewed sensitive resources
	permission.ResourceOrganization:          "organization settings hold no secret; sign-in settings have their own resources and the address printed on invoices is a payment destination",
	permission.ResourceBusinessUnit:          "business unit details",
	permission.ResourceTableConfiguration:    "saved table views",
	permission.ResourceCustomFieldDefinition: "custom field definitions",
	permission.ResourceSequenceConfig:        "number sequences",
	permission.ResourceEmailLog:              "what was sent, never a credential",
	permission.ResourceEmailSuppression:      "suppressed addresses",
	permission.ResourceTableChangeAlert:      "table change alerts",
	permission.ResourceAIUsage:               "model call counts, tokens and estimated cost; never a provider credential or endpoint",
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

type PaymentDestination struct {
	Table    string
	Columns  []string
	OnInsert bool
	Limit    Limit
}

var paymentDestinations = []PaymentDestination{
	{
		Table:   "invoices",
		Columns: []string{"remittance_instructions", "factoring_override"},
		Limit:   LimitPaymentDestinations,
	},
	{
		Table:   "billing_controls",
		Columns: []string{"default_invoice_terms"},
		Limit:   LimitPaymentDestinations,
	},
	{
		Table:   "organizations",
		Columns: []string{"address_line1", "address_line2", "city", "state_id", "postal_code"},
		Limit:   LimitPaymentDestinations,
	},
	{
		Table: "carriers",
		Columns: []string{
			"remit_to_name", "remit_address_line_1", "remit_address_line_2", "remit_city",
			"remit_state_id", "remit_postal_code",
		},
		OnInsert: true,
		Limit:    LimitPaymentDestinations,
	},
	{
		Table:    "customer_billing_profiles",
		Columns:  []string{"factoring_company_id"},
		OnInsert: true,
		Limit:    LimitPaymentDestinations,
	},
	{
		Table: "factoring_companies",
		Columns: []string{
			"address_line_1", "address_line_2", "city", "state_id", "postal_code",
			"remittance_instructions", "notice_of_assignment",
		},
		OnInsert: true,
		Limit:    LimitFactoring,
	},
}

func PaymentDestinations() []PaymentDestination {
	out := make([]PaymentDestination, len(paymentDestinations))
	for i, d := range paymentDestinations {
		d.Columns = slices.Clone(d.Columns)
		out[i] = d
	}
	return out
}

func ErrPaymentDestinationRefused() *errortypes.AuthorizationError {
	return errortypes.NewAuthorizationError(
		"Trenova support can never change where money is sent: remittance instructions, remit-to addresses, the factor an invoice or customer pays, or your company address printed on invoices",
	)
}

func ErrModelCallRefused() *errortypes.BusinessError {
	return errortypes.NewBusinessError(
		"Trenova support cannot use the assistant here. Each assistant answer is paid for by your organization's AI providers, so an administrator has to allow it when granting access",
	)
}

func ErrReadOnlySession() *errortypes.AuthorizationError {
	return errortypes.NewAuthorizationError(
		"This Trenova support session is read-only, so it cannot change anything. An administrator of this organization can grant full access instead",
	)
}

var AssistantOperations = []permission.Operation{permission.OpRead, permission.OpCreate}

// SensitiveCategories are the role editor categories whose every resource must be classified.
var SensitiveCategories = []string{"Users & security", "Organization settings", "AI & agents"}

var sensitiveNameParts = []string{
	"integration", "mailbox", "provider", "api_key", "credential", "scim", "factoring",
	"remittance", "portal", "device", "protected_", "support",
}

// SensitiveResource reports whether a resource must be named in the support policy.
func SensitiveResource(resource, category string) bool {
	if slices.Contains(SensitiveCategories, category) {
		return true
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

// DeniedLimit is why no support session may use the resource as op, or zero when the policy
// does not refuse it.
func DeniedLimit(resource string, op permission.Operation) Limit {
	if op == permission.OpExport {
		return LimitExport
	}
	r, ok := denied[permission.Resource(resource)]
	if !ok {
		return 0
	}
	if r.denial == denyAll || op != permission.OpRead {
		return r.limit
	}
	return 0
}

// Allows is the rule on the support principal: whether a session under a grant of scope may
// perform op on resource. Every non-read operation is refused to a read-only grant, the deny
// list is refused to every grant, and the assistant is left to the grant's own choice.
func Allows(scope Scope, resource string, op permission.Operation) bool {
	if !scope.IsValid() {
		return false
	}
	if permission.Resource(resource) == permission.ResourceAssistant {
		return false
	}
	if DeniedLimit(resource, op) != 0 {
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
// child operation is taken off its parent too.
func Effective(
	scope Scope,
	defs []*permissionspec.ResourceDefinition,
) map[string][]permission.Operation {
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
			out[def.ParentResource] = slices.DeleteFunc(
				out[def.ParentResource],
				func(held permission.Operation) bool {
					return held == op.Operation
				},
			)
		}
	}
	for resource, ops := range out {
		if len(ops) == 0 {
			delete(out, resource)
		}
	}
	return out
}
