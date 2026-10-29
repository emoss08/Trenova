package factoringports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgreementRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type ListAgreementsRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
}

type ListAgreementsByFactoringCompanyIDsRequest struct {
	TenantInfo          pagination.TenantInfo `json:"-"`
	FactoringCompanyIDs []pulid.ID            `json:"factoringCompanyIds"`
}

type GetAgreementInForceRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
	At                 int64                 `json:"at"`
}

type ListAgreementsInForceRequest struct {
	TenantInfo          pagination.TenantInfo `json:"-"`
	FactoringCompanyIDs []pulid.ID            `json:"factoringCompanyIds"`
	At                  int64                 `json:"at"`
}

type AgreementRepository interface {
	Create(ctx context.Context, entity *factoring.Agreement) (*factoring.Agreement, error)
	Update(ctx context.Context, entity *factoring.Agreement) (*factoring.Agreement, error)
	GetByID(ctx context.Context, req GetAgreementRequest) (*factoring.Agreement, error)
	ListByFactoringCompany(
		ctx context.Context,
		req ListAgreementsRequest,
	) ([]*factoring.Agreement, error)
	ListByFactoringCompanyIDs(
		ctx context.Context,
		req *ListAgreementsByFactoringCompanyIDsRequest,
	) (map[pulid.ID][]*factoring.Agreement, error)
	GetInForce(ctx context.Context, req GetAgreementInForceRequest) (*factoring.Agreement, error)
	ListInForce(
		ctx context.Context,
		req *ListAgreementsInForceRequest,
	) (map[pulid.ID]*factoring.Agreement, error)
}

type GetNoaRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type GetOpenNoaRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	CustomerID         pulid.ID              `json:"customerId"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
}

type NoaPair struct {
	CustomerID         pulid.ID `json:"customerId"`
	FactoringCompanyID pulid.ID `json:"factoringCompanyId"`
}

type ListOpenNoasRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Pairs      []NoaPair             `json:"pairs"`
}

type ListNoaConnectionRequest struct {
	Filter             *pagination.QueryOptions `json:"filter"`
	Cursor             pagination.CursorInfo    `json:"-"`
	CustomerID         pulid.ID                 `json:"customerId"`
	FactoringCompanyID pulid.ID                 `json:"factoringCompanyId"`
	Statuses           []factoring.NoaStatus    `json:"statuses"`
}

type ListExpiringNoasRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	At         int64                 `json:"at"`
	Before     int64                 `json:"before"`
	Limit      int                   `json:"limit"`
}

type ClaimNoaExpiryWarningRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
	At         int64                 `json:"at"`
}

type NoaRepository interface {
	Create(ctx context.Context, entity *factoring.Noa) (*factoring.Noa, error)
	Update(ctx context.Context, entity *factoring.Noa) (*factoring.Noa, error)
	GetByID(ctx context.Context, req GetNoaRequest) (*factoring.Noa, error)
	GetOpen(ctx context.Context, req *GetOpenNoaRequest) (*factoring.Noa, error)
	ListOpen(ctx context.Context, req *ListOpenNoasRequest) (map[NoaPair]*factoring.Noa, error)
	ListConnection(
		ctx context.Context,
		req *ListNoaConnectionRequest,
	) (*pagination.CursorListResult[*factoring.Noa], error)
	ListExpiring(ctx context.Context, req ListExpiringNoasRequest) ([]*factoring.Noa, error)
	ClaimExpiryWarning(ctx context.Context, req ClaimNoaExpiryWarningRequest) (bool, error)
}

type GetInvoiceStateRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"invoiceId"`
}

type ListInvoiceStatesByInvoiceIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"invoiceIds"`
}

type ListInvoiceStateConnectionRequest struct {
	Filter             *pagination.QueryOptions     `json:"filter"`
	Cursor             pagination.CursorInfo        `json:"-"`
	Verdicts           []factoringreadiness.Verdict `json:"verdicts"`
	FactoringCompanyID pulid.ID                     `json:"factoringCompanyId"`
	CustomerID         pulid.ID                     `json:"customerId"`
}

type ClaimReadyAnnouncementRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"invoiceId"`
	ReadySince int64                 `json:"readySince"`
	At         int64                 `json:"at"`
}

type ListReadyUnsubmittedRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	ReadyBefore int64                 `json:"readyBefore"`
	Limit       int                   `json:"limit"`
}

type ListCandidateInvoicesRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	AfterInvoiceID pulid.ID              `json:"afterInvoiceId"`
	Limit          int                   `json:"limit"`
}

type ListCustomerInvoiceIDsRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	CustomerIDs        []pulid.ID            `json:"customerIds"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
	Limit              int                   `json:"limit"`
}

type GetSummaryRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	ReadyBefore      int64                 `json:"readyBefore"`
	ChargebackBefore int64                 `json:"chargebackBefore"`
}

type VerdictTotals struct {
	Count       int   `json:"count"`
	AmountMinor int64 `json:"amountMinor"`
}

type Summary struct {
	Ready            VerdictTotals `json:"ready"`
	NotReady         VerdictTotals `json:"notReady"`
	Submitted        VerdictTotals `json:"submitted"`
	Funded           VerdictTotals `json:"funded"`
	ReserveHeld      VerdictTotals `json:"reserveHeld"`
	ChargebacksDue   VerdictTotals `json:"chargebacksDue"`
	ReadyUnsubmitted VerdictTotals `json:"readyUnsubmitted"`
}

type ShipmentContact struct {
	ShipmentID   pulid.ID `json:"shipmentId"`
	DriverName   string   `json:"driverName"`
	DriverEmail  string   `json:"driverEmail"`
	CarrierName  string   `json:"carrierName"`
	CarrierEmail string   `json:"carrierEmail"`
}

type ListShipmentContactsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	ShipmentIDs []pulid.ID            `json:"shipmentIds"`
}

type SetInvoiceOverrideRequest struct {
	TenantInfo pagination.TenantInfo     `json:"-"`
	InvoiceID  pulid.ID                  `json:"invoiceId"`
	Override   invoice.FactoringOverride `json:"override"`
	Reason     string                    `json:"reason"`

	FactoringCompanyID   *pulid.ID `json:"factoringCompanyId"`
	FactoringAgreementID *pulid.ID `json:"factoringAgreementId"`
}

type InvoiceStateRepository interface {
	SetInvoiceOverride(ctx context.Context, req *SetInvoiceOverrideRequest) error
	ListSweepTenants(ctx context.Context) ([]pagination.TenantInfo, error)
	ListShipmentContacts(
		ctx context.Context,
		req *ListShipmentContactsRequest,
	) (map[pulid.ID]*ShipmentContact, error)
	Get(ctx context.Context, req GetInvoiceStateRequest) (*factoringreadiness.InvoiceState, error)
	ListByInvoiceIDs(
		ctx context.Context,
		req *ListInvoiceStatesByInvoiceIDsRequest,
	) (map[pulid.ID]*factoringreadiness.InvoiceState, error)
	Upsert(ctx context.Context, entity *factoringreadiness.InvoiceState) error
	ListConnection(
		ctx context.Context,
		req *ListInvoiceStateConnectionRequest,
	) (*pagination.CursorListResult[*factoringreadiness.InvoiceState], error)
	ClaimReadyAnnouncement(ctx context.Context, req *ClaimReadyAnnouncementRequest) (bool, error)
	ListReadyUnsubmitted(
		ctx context.Context,
		req ListReadyUnsubmittedRequest,
	) ([]*factoringreadiness.InvoiceState, error)
	ListCandidateInvoiceIDs(
		ctx context.Context,
		req ListCandidateInvoicesRequest,
	) ([]pulid.ID, error)
	ListCustomerInvoiceIDs(
		ctx context.Context,
		req *ListCustomerInvoiceIDsRequest,
	) ([]pulid.ID, error)
	GetSummary(ctx context.Context, req GetSummaryRequest) (*Summary, error)
}
