package responseguard

import (
	"reflect"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentparsingrule"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/shared/stringutils"
)

const domainPackagePrefix = "github.com/emoss08/trenova/internal/modules/"

// namedKinds are the domain types whose Go name does not spell their
// resource. Every other domain struct named for a registered resource
// (customer.Customer, worker.Worker, shipment.ShipmentMove) is that resource
// with no entry here; a type named for none is part of the record that carries
// it. The relation test fails on a relation whose type is neither.
var namedKinds = map[reflect.Type]permission.Resource{
	reflect.TypeFor[shipment.Stop]():                       permission.ResourceShipmentStop,
	reflect.TypeFor[customerpayment.Payment]():             permission.ResourceCustomerPayment,
	reflect.TypeFor[billingqueue.BillingQueueItem]():       permission.ResourceBillingQueue,
	reflect.TypeFor[factoring.Agreement]():                 permission.ResourceFactoringAgreement,
	reflect.TypeFor[factoring.Noa]():                       permission.ResourceFactoringNoa,
	reflect.TypeFor[factoring.Submission]():                permission.ResourceFactoringSubmission,
	reflect.TypeFor[factoring.ReserveEntry]():              permission.ResourceFactoringReserve,
	reflect.TypeFor[factoring.Remittance]():                permission.ResourceFactoringRemittance,
	reflect.TypeFor[rateagreement.RateAgreementVersion]():  permission.ResourceRateAgreement,
	reflect.TypeFor[report.ReportDefinition]():             permission.ResourceReport,
	reflect.TypeFor[servicefailure.ReasonCode]():           permission.ResourceServiceFailureReasonCode,
	reflect.TypeFor[edi.EDIPartner]():                      permission.ResourceEDI,
	reflect.TypeFor[email.Profile]():                       permission.ResourceEmailProfile,
	reflect.TypeFor[email.Message]():                       permission.ResourceEmailLog,
	reflect.TypeFor[glaccount.GLAccount]():                 permission.ResourceGeneralLedgerAccount,
	reflect.TypeFor[documentparsingrule.RuleSet]():         permission.ResourceDocumentParsingRule,
	reflect.TypeFor[worker.WorkerLeaveCase]():              permission.ResourceWorkerLeave,
	reflect.TypeFor[worker.WorkerLeaveEntry]():             permission.ResourceWorkerLeave,
	reflect.TypeFor[worker.WorkerDOTViolation]():           permission.ResourceWorkerDOTTest,
	reflect.TypeFor[worker.WorkerClearinghouseQuery]():     permission.ResourceWorkerDOTTest,
	reflect.TypeFor[worker.OSHAAnnualSummary]():            permission.ResourceWorkerInjury,
	reflect.TypeFor[worker.TimeClockEntry]():               permission.ResourceTimesheet,
	reflect.TypeFor[worker.PayrollExport]():                permission.ResourceTimesheet,
	reflect.TypeFor[worker.WorkerProfileChangeRequest]():   permission.ResourceProfileChangeRequest,
	reflect.TypeFor[driverpay.WorkerBenefitEnrollment]():   permission.ResourceBenefitPlan,
	reflect.TypeFor[worker.WorkerEmploymentVerification](): permission.ResourceQualification,
}

// kindTable maps a Go type to the kind of record it holds.
type kindTable struct {
	registry *permission.Registry

	mu    sync.RWMutex
	kinds map[reflect.Type]permission.Resource
}

func newKindTable(registry *permission.Registry) *kindTable {
	return &kindTable{
		registry: registry,
		kinds:    make(map[reflect.Type]permission.Resource, len(namedKinds)+32),
	}
}

func (k *kindTable) kindOf(t reflect.Type) (permission.Resource, bool) {
	if t.Kind() != reflect.Struct {
		return "", false
	}
	k.mu.RLock()
	kind, ok := k.kinds[t]
	k.mu.RUnlock()
	if ok {
		return kind, kind != ""
	}

	kind = resolveKind(k.registry, t)

	k.mu.Lock()
	k.kinds[t] = kind
	k.mu.Unlock()

	return kind, kind != ""
}

var namedKindsByName = func() map[string]permission.Resource {
	out := make(map[string]permission.Resource, len(namedKinds))
	for t, kind := range namedKinds {
		out[t.PkgPath()+"."+t.Name()] = kind
	}
	return out
}()

func resolveKind(registry *permission.Registry, t reflect.Type) permission.Resource {
	return kindOfName(registry, t.PkgPath(), t.Name())
}

// kindOfName is the kind of the domain type a package path and type name
// name, so a tool reading source agrees with what a response is gated as.
func kindOfName(registry *permission.Registry, pkgPath, name string) permission.Resource {
	if kind, ok := namedKindsByName[pkgPath+"."+name]; ok {
		return kind
	}
	if name == "" || !isDomainPackage(pkgPath) {
		return ""
	}
	if resource := stringutils.ConvertCamelToSnake(name); registry.HasResource(resource) {
		return permission.Resource(resource)
	}

	return ""
}

func isDomainPackage(path string) bool {
	rest, ok := strings.CutPrefix(path, domainPackagePrefix)
	return ok && strings.Contains(rest, "/domain/")
}
