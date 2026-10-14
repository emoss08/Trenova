package factoringports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgreementInput struct {
	FactoringCompanyID   pulid.ID                      `json:"factoringCompanyId"`
	EffectiveFrom        int64                         `json:"effectiveFrom"`
	EffectiveTo          *int64                        `json:"effectiveTo"`
	AdvanceRateBP        int                           `json:"advanceRateBp"`
	ReserveRateBP        int                           `json:"reserveRateBp"`
	MinimumFeeMinor      int64                         `json:"minimumFeeMinor"`
	ChargebackWindowDays int                           `json:"chargebackWindowDays"`
	Recourse             factoring.Recourse            `json:"recourse"`
	AccountingTreatment  factoring.AccountingTreatment `json:"accountingTreatment"`
	FundingDays          int                           `json:"fundingDays"`
	SubmissionTime       int                           `json:"submissionTime"`
	Notes                string                        `json:"notes"`
	FeeTiers             []*factoring.FeeTier          `json:"feeTiers"`
	DocumentTypeIDs      []pulid.ID                    `json:"documentTypeIds"`
}

type CreateAgreementRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Input      AgreementInput        `json:"input"`
}

type UpdateAgreementRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
	Version    int64                 `json:"version"`
	Input      AgreementInput        `json:"input"`
}

type ChangeAgreementStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
	Version    int64                 `json:"version"`
	EndAt      *int64                `json:"endAt"`
}

type AgreementService interface {
	List(ctx context.Context, req ListAgreementsRequest) ([]*factoring.Agreement, error)
	Get(ctx context.Context, req GetAgreementRequest) (*factoring.Agreement, error)
	Create(
		ctx context.Context,
		req *CreateAgreementRequest,
		actor *permissionports.RequestActor,
	) (*factoring.Agreement, error)
	Update(
		ctx context.Context,
		req *UpdateAgreementRequest,
		actor *permissionports.RequestActor,
	) (*factoring.Agreement, error)
	Activate(
		ctx context.Context,
		req *ChangeAgreementStatusRequest,
		actor *permissionports.RequestActor,
	) (*factoring.Agreement, error)
	End(
		ctx context.Context,
		req *ChangeAgreementStatusRequest,
		actor *permissionports.RequestActor,
	) (*factoring.Agreement, error)
}

type DraftNoaRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	CustomerID         pulid.ID              `json:"customerId"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
	Notes              string                `json:"notes"`
}

type SendNoaRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
	Version    int64                 `json:"version"`
	Recipient  string                `json:"recipient"`
}

type AcknowledgeNoaRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	ID               pulid.ID              `json:"id"`
	Version          int64                 `json:"version"`
	AcknowledgedAt   int64                 `json:"acknowledgedAt"`
	SignedDocumentID pulid.ID              `json:"signedDocumentId"`
	ExpiresAt        *int64                `json:"expiresAt"`
}

type ReleaseNoaRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
	Version    int64                 `json:"version"`
	Reason     string                `json:"reason"`
}

type MessagePlan struct {
	To             []string  `json:"to"`
	CC             []string  `json:"cc"`
	BCC            []string  `json:"bcc"`
	Subject        string    `json:"subject"`
	HTML           string    `json:"html"`
	Text           string    `json:"text"`
	AttachmentName string    `json:"attachmentName"`
	VersionID      *pulid.ID `json:"versionId"`
}

type NoaSendPlan struct {
	Noa     *factoring.Noa `json:"noa"`
	Message MessagePlan    `json:"message"`
}

type DocumentRequestPlan struct {
	Invoice       *invoice.Invoice `json:"invoice"`
	RecipientKind string           `json:"recipientKind"`
	DocumentTypes []string         `json:"documentTypes"`
	ShipmentIDs   []pulid.ID       `json:"shipmentIds"`
	Message       MessagePlan      `json:"message"`
}

type OverridePlan struct {
	Before  *invoice.Invoice `json:"before"`
	After   *invoice.Invoice `json:"after"`
	Changed bool             `json:"changed"`
}

type RenderedLetter struct {
	FileName string `json:"fileName"`
	PDF      []byte `json:"-"`
}

type NoaService interface {
	RenderLetter(ctx context.Context, req GetNoaRequest) (*RenderedLetter, error)
	PlanSend(ctx context.Context, req *SendNoaRequest, userID pulid.ID) (*NoaSendPlan, error)
	List(
		ctx context.Context,
		req *ListNoaConnectionRequest,
	) (*pagination.CursorListResult[*factoring.Noa], error)
	Get(ctx context.Context, req GetNoaRequest) (*factoring.Noa, error)
	Draft(
		ctx context.Context,
		req *DraftNoaRequest,
		actor *permissionports.RequestActor,
	) (*factoring.Noa, error)
	Send(
		ctx context.Context,
		req *SendNoaRequest,
		actor *permissionports.RequestActor,
	) (*factoring.Noa, error)
	Acknowledge(
		ctx context.Context,
		req *AcknowledgeNoaRequest,
		actor *permissionports.RequestActor,
	) (*factoring.Noa, error)
	Release(
		ctx context.Context,
		req *ReleaseNoaRequest,
		actor *permissionports.RequestActor,
	) (*factoring.Noa, error)
}

type GetReadinessRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"invoiceId"`
}

type EvaluateInvoicesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"invoiceIds"`
}

type EvaluateResult struct {
	Evaluated   int `json:"evaluated"`
	BecameReady int `json:"becameReady"`
	LeftReady   int `json:"leftReady"`
	Announced   int `json:"announced"`
	Failed      int `json:"failed"`
}

func (r *EvaluateResult) Add(other *EvaluateResult) {
	if other == nil {
		return
	}
	r.Evaluated += other.Evaluated
	r.BecameReady += other.BecameReady
	r.LeftReady += other.LeftReady
	r.Announced += other.Announced
	r.Failed += other.Failed
}

type SweepResult struct {
	EvaluateResult
	NoasWarned        int `json:"noasWarned"`
	ChargebacksWarned int `json:"chargebacksWarned"`
}

type SetOverrideRequest struct {
	TenantInfo pagination.TenantInfo     `json:"-"`
	InvoiceID  pulid.ID                  `json:"invoiceId"`
	Override   invoice.FactoringOverride `json:"override"`
	Reason     string                    `json:"reason"`
}

type RequestMissingDocumentsRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	InvoiceID       pulid.ID              `json:"invoiceId"`
	DocumentTypeIDs []pulid.ID            `json:"documentTypeIds"`
	Recipient       string                `json:"recipient"`
	Message         string                `json:"message"`
}

type RequestMissingDocumentsResult struct {
	Recipient     string     `json:"recipient"`
	RecipientKind string     `json:"recipientKind"`
	DocumentTypes []string   `json:"documentTypes"`
	ShipmentIDs   []pulid.ID `json:"shipmentIds"`
	Sent          bool       `json:"sent"`
}

type ReadinessService interface {
	GetReadiness(ctx context.Context, req GetReadinessRequest) (*factoring.InvoiceState, error)
	Evaluate(ctx context.Context, req *EvaluateInvoicesRequest) (*EvaluateResult, error)
	ListStates(
		ctx context.Context,
		req *ListInvoiceStateConnectionRequest,
	) (*pagination.CursorListResult[*factoring.InvoiceState], error)
	GetStates(
		ctx context.Context,
		req *ListInvoiceStatesByInvoiceIDsRequest,
	) (map[pulid.ID]*factoring.InvoiceState, error)
	Summary(ctx context.Context, tenantInfo pagination.TenantInfo) (*Summary, error)
	SetOverride(
		ctx context.Context,
		req *SetOverrideRequest,
		actor *permissionports.RequestActor,
	) (*factoring.InvoiceState, error)
	PlanOverride(ctx context.Context, req *SetOverrideRequest) (*OverridePlan, error)
	RequestMissingDocuments(
		ctx context.Context,
		req *RequestMissingDocumentsRequest,
		actor *permissionports.RequestActor,
	) (*RequestMissingDocumentsResult, error)
	PlanDocumentRequest(
		ctx context.Context,
		req *RequestMissingDocumentsRequest,
		userID pulid.ID,
	) (*DocumentRequestPlan, error)
	ListSweepTenants(ctx context.Context) ([]pagination.TenantInfo, error)
	SweepTenant(ctx context.Context, tenantInfo pagination.TenantInfo) (*SweepResult, error)
}

type EvaluationTargets struct {
	InvoiceIDs          []pulid.ID `json:"invoiceIds,omitempty"`
	CustomerIDs         []pulid.ID `json:"customerIds,omitempty"`
	FactoringCompanyIDs []pulid.ID `json:"factoringCompanyIds,omitempty"`
}

func (t *EvaluationTargets) Empty() bool {
	return t == nil ||
		len(t.InvoiceIDs) == 0 && len(t.CustomerIDs) == 0 && len(t.FactoringCompanyIDs) == 0
}

type EvaluationRequester interface {
	RequestEvaluation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		targets *EvaluationTargets,
	)
}
