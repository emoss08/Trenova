package volumegen

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/shared/pulid"
)

// FactorPeriod is a span in which a customer's invoices were assigned to one
// factor: billing stamps an invoice with the factor whose period covers the
// moment it posted, as invoicepostingservice.assignFactoring does.
type FactorPeriod struct {
	Factor *Factor
	From   int64
	To     int64
}

func (p *FactorPeriod) Covers(at int64) bool {
	return at >= p.From && (p.To == 0 || at < p.To)
}

type FactorStyle struct {
	Headers    []string
	Ext        string
	Purchase   string
	Release    string
	Decline    string
	Chargeback string
	Other      string
}

type Factor struct {
	Company   *factoringcompany.FactoringCompany
	Agreement *factoring.Agreement
	Transport *factoringsubmission.Transport
	Noas      map[pulid.ID]*factoring.Noa
	Style     FactorStyle
	Slug      string
	Profiled  bool
}

type FactorState struct {
	Factors []*Factor
	ByID    map[pulid.ID]*Factor
}
