package factoring

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const ReadyUnsubmittedSeconds = 24 * 60 * 60

// Whether each factored invoice is ready to submit to its factor, what is holding it back, and since when
type InvoiceState struct {
	keel.Table `keel:"factoring_invoice_states,report=Billing,label=Invoice Factoring State"`

	InvoiceID          pulid.ID  `json:"id"                 keel:"invoice_id,pk"`
	OrganizationID     pulid.ID  `json:"organizationId"     keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID  `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	FactoringCompanyID *pulid.ID `json:"factoringCompanyId" keel:"factoring_company_id"`
	CustomerID         pulid.ID  `json:"customerId"         keel:"customer_id"`
	Verdict            Verdict   `json:"verdict"            keel:"verdict"`
	Checks             []Check   `json:"checks"             keel:"checks,jsonb,noreport"`
	FailedChecks       int       `json:"failedChecks"       keel:"failed_checks,format=count"`
	OpenBalanceMinor   int64     `json:"openBalanceMinor"   keel:"open_balance_minor,label=Open Balance (Minor)"`
	ReadySince         *int64    `json:"readySince"         keel:"ready_since,type=epoch"`
	ReadyAnnouncedAt   *int64    `json:"readyAnnouncedAt"   keel:"ready_announced_at"`
	EvaluatedAt        int64     `json:"evaluatedAt"        keel:"evaluated_at"`
	CreatedAt          int64     `json:"createdAt"          keel:"created_at"`

	Invoice          *invoice.Invoice                   `json:"invoice,omitempty"          keel:"belongs:invoice_id"`
	Customer         *customer.Customer                 `json:"customer,omitempty"         keel:"belongs:customer_id"`
	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" keel:"belongs:factoring_company_id"`
}

type StateTransition struct {
	BecameReady bool
	LeftReady   bool
}

func NextState(
	previous *InvoiceState,
	next *InvoiceState,
	readiness *Readiness,
	at int64,
) StateTransition {
	next.Verdict = readiness.Verdict
	next.Checks = readiness.Checks
	next.FailedChecks = readiness.FailedCount()
	next.EvaluatedAt = at
	next.CreatedAt = at
	if previous != nil && previous.CreatedAt > 0 {
		next.CreatedAt = previous.CreatedAt
	}

	wasReady := previous != nil && previous.Verdict == VerdictReady
	if readiness.Verdict != VerdictReady {
		next.ReadySince = nil
		next.ReadyAnnouncedAt = nil
		return StateTransition{LeftReady: wasReady}
	}
	if wasReady && previous.ReadySince != nil {
		since := *previous.ReadySince
		next.ReadySince = &since
		next.ReadyAnnouncedAt = previous.ReadyAnnouncedAt
		return StateTransition{}
	}
	since := at
	next.ReadySince = &since
	next.ReadyAnnouncedAt = nil
	return StateTransition{BecameReady: true}
}

func (s *InvoiceState) ReadyLongerThan(at, seconds int64) bool {
	return s != nil && s.Verdict == VerdictReady && s.ReadySince != nil &&
		at-*s.ReadySince >= seconds
}

func (s *InvoiceState) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "facst",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "verdict", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (s *InvoiceState) GetID() pulid.ID             { return s.InvoiceID }
func (s *InvoiceState) GetCreatedAt() int64         { return s.CreatedAt }
func (s *InvoiceState) GetOrganizationID() pulid.ID { return s.OrganizationID }
func (s *InvoiceState) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }
func (s *InvoiceState) GetTableName() string        { return "factoring_invoice_states" }
