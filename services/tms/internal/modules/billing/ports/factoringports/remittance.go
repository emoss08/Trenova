package factoringports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetRemittanceRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	ID           pulid.ID              `json:"id"`
	IncludeLines bool                  `json:"includeLines"`
	ForUpdate    bool                  `json:"-"`
}

type ListRemittanceConnectionRequest struct {
	Filter             *pagination.QueryOptions     `json:"filter"`
	Cursor             pagination.CursorInfo        `json:"-"`
	FactoringCompanyID pulid.ID                     `json:"factoringCompanyId"`
	Statuses           []factoring.RemittanceStatus `json:"statuses"`
}

type ListRemittanceLineConnectionRequest struct {
	Filter        *pagination.QueryOptions          `json:"filter"`
	TenantInfo    pagination.TenantInfo             `json:"-"`
	Cursor        pagination.CursorInfo             `json:"-"`
	RemittanceID  pulid.ID                          `json:"remittanceId"`
	MatchStatuses []factoring.RemittanceMatchStatus `json:"matchStatuses"`
}

type ListRemittanceCandidatesRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
	Invoices           []string              `json:"invoices"`
	Amounts            []int64               `json:"amounts"`
	ItemIDs            []pulid.ID            `json:"itemIds"`
	Limit              int                   `json:"limit"`
}

type FindRemittanceByDocumentRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	InboundMessageID pulid.ID              `json:"inboundMessageId"`
	DocumentID       pulid.ID              `json:"documentId"`
}

type GetRemittanceProfileRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
}

type ListOpenRemittancesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Limit      int                   `json:"limit"`
}

type OpenRemittance struct {
	RemittanceID       pulid.ID                   `json:"remittanceId"       bun:"remittance_id"`
	FactoringCompanyID pulid.ID                   `json:"factoringCompanyId" bun:"factoring_company_id"`
	FactorName         string                     `json:"factorName"         bun:"factor_name"`
	FileName           string                     `json:"fileName"           bun:"file_name"`
	Status             factoring.RemittanceStatus `json:"status"             bun:"status"`
	ExceptionCount     int                        `json:"exceptionCount"     bun:"exception_count"`
	MatchedCount       int                        `json:"matchedCount"       bun:"matched_count"`
	CreatedAt          int64                      `json:"createdAt"          bun:"created_at"`
}

type RemittanceRepository interface {
	Create(ctx context.Context, entity *factoring.Remittance) (*factoring.Remittance, error)
	Update(ctx context.Context, entity *factoring.Remittance) (*factoring.Remittance, error)
	Get(ctx context.Context, req GetRemittanceRequest) (*factoring.Remittance, error)
	FindByInboundDocument(
		ctx context.Context,
		req *FindRemittanceByDocumentRequest,
	) (*factoring.Remittance, error)
	ListConnection(
		ctx context.Context,
		req *ListRemittanceConnectionRequest,
	) (*pagination.CursorListResult[*factoring.Remittance], error)
	ListLineConnection(
		ctx context.Context,
		req *ListRemittanceLineConnectionRequest,
	) (*pagination.CursorListResult[*factoring.RemittanceLine], error)
	ReplaceLines(
		ctx context.Context,
		remittance *factoring.Remittance,
	) error
	UpdateLines(ctx context.Context, lines []*factoring.RemittanceLine) error
	ListCandidates(
		ctx context.Context,
		req *ListRemittanceCandidatesRequest,
	) ([]*factoring.RemittanceCandidate, error)
	ListOpen(ctx context.Context, req *ListOpenRemittancesRequest) ([]*OpenRemittance, error)
	GetProfile(
		ctx context.Context,
		req GetRemittanceProfileRequest,
	) (*factoring.RemittanceProfile, error)
	SaveProfile(ctx context.Context, profile *factoring.RemittanceProfile) error
}
