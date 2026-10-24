package factoring

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const ReadyUnsubmittedSeconds = 24 * 60 * 60

// Whether each factored invoice is ready to submit to its factor, what is holding it back, and since when
type InvoiceState struct {
	bun.BaseModel `bun:"table:factoring_invoice_states,alias:facst" json:"-"`
	keel.Table    `                                                          keel:"factoring_invoice_states,report=Billing,label=Invoice Factoring State"`

	InvoiceID          pulid.ID  `json:"id"                 bun:"invoice_id,pk,type:VARCHAR(100),notnull"         keel:"invoice_id,pk"`
	OrganizationID     pulid.ID  `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"    keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID  `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"   keel:"business_unit_id,pk,tenant"`
	FactoringCompanyID *pulid.ID `json:"factoringCompanyId" bun:"factoring_company_id,type:VARCHAR(100),nullzero" keel:"factoring_company_id"`
	CustomerID         pulid.ID  `json:"customerId"         bun:"customer_id,type:VARCHAR(100),notnull"           keel:"customer_id"`
	Verdict            Verdict   `json:"verdict"            bun:"verdict,type:VARCHAR(20),notnull"                keel:"verdict"`
	Checks             []Check   `json:"checks"             bun:"checks,type:JSONB,notnull"                       keel:"checks,jsonb,noreport"`
	FailedChecks       int       `json:"failedChecks"       bun:"failed_checks,type:SMALLINT,notnull"             keel:"failed_checks,format=count"`
	OpenBalanceMinor   int64     `json:"openBalanceMinor"   bun:"open_balance_minor,type:BIGINT,notnull"          keel:"open_balance_minor,label=Open Balance (Minor)"`
	ReadySince         *int64    `json:"readySince"         bun:"ready_since,type:BIGINT,nullzero"                keel:"ready_since,type=epoch"`
	ReadyAnnouncedAt   *int64    `json:"readyAnnouncedAt"   bun:"ready_announced_at,type:BIGINT,nullzero"         keel:"ready_announced_at"`
	EvaluatedAt        int64     `json:"evaluatedAt"        bun:"evaluated_at,type:BIGINT,notnull"                keel:"evaluated_at"`
	CreatedAt          int64     `json:"createdAt"          bun:"created_at,type:BIGINT,notnull"                  keel:"created_at"`

	Invoice          *invoice.Invoice                   `json:"invoice,omitempty"          bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"           keel:"belongs:invoice_id"`
	Customer         *customer.Customer                 `json:"customer,omitempty"         bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"          keel:"belongs:customer_id"`
	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" bun:"rel:belongs-to,join:factoring_company_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:factoring_company_id"`
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
