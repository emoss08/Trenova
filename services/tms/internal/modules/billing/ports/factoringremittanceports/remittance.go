package factoringremittanceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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
	Filter             *pagination.QueryOptions               `json:"filter"`
	Cursor             pagination.CursorInfo                  `json:"-"`
	FactoringCompanyID pulid.ID                               `json:"factoringCompanyId"`
	InboundMessageID   pulid.ID                               `json:"inboundMessageId"`
	Statuses           []factoringremittance.RemittanceStatus `json:"statuses"`
}

type ListRemittanceLineConnectionRequest struct {
	Filter        *pagination.QueryOptions                    `json:"filter"`
	TenantInfo    pagination.TenantInfo                       `json:"-"`
	Cursor        pagination.CursorInfo                       `json:"-"`
	RemittanceID  pulid.ID                                    `json:"remittanceId"`
	MatchStatuses []factoringremittance.RemittanceMatchStatus `json:"matchStatuses"`
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
	RemittanceID       pulid.ID                             `json:"remittanceId"`
	FactoringCompanyID pulid.ID                             `json:"factoringCompanyId"`
	FactorName         string                               `json:"factorName"`
	FileName           string                               `json:"fileName"`
	Status             factoringremittance.RemittanceStatus `json:"status"`
	ExceptionCount     int                                  `json:"exceptionCount"`
	MatchedCount       int                                  `json:"matchedCount"`
	CreatedAt          int64                                `json:"createdAt"`
}

type RemittanceRepository interface {
	Create(
		ctx context.Context,
		entity *factoringremittance.Remittance,
	) (*factoringremittance.Remittance, error)
	Update(
		ctx context.Context,
		entity *factoringremittance.Remittance,
	) (*factoringremittance.Remittance, error)
	Get(ctx context.Context, req GetRemittanceRequest) (*factoringremittance.Remittance, error)
	FindByInboundDocument(
		ctx context.Context,
		req *FindRemittanceByDocumentRequest,
	) (*factoringremittance.Remittance, error)
	ListConnection(
		ctx context.Context,
		req *ListRemittanceConnectionRequest,
	) (*pagination.CursorListResult[*factoringremittance.Remittance], error)
	ListLineConnection(
		ctx context.Context,
		req *ListRemittanceLineConnectionRequest,
	) (*pagination.CursorListResult[*factoringremittance.RemittanceLine], error)
	ReplaceLines(
		ctx context.Context,
		remittance *factoringremittance.Remittance,
	) error
	UpdateLines(ctx context.Context, lines []*factoringremittance.RemittanceLine) error
	ListCandidates(
		ctx context.Context,
		req *ListRemittanceCandidatesRequest,
	) ([]*factoringremittance.RemittanceCandidate, error)
	ListOpen(ctx context.Context, req *ListOpenRemittancesRequest) ([]*OpenRemittance, error)
	GetProfile(
		ctx context.Context,
		req GetRemittanceProfileRequest,
	) (*factoringremittance.RemittanceProfile, error)
	SaveProfile(ctx context.Context, profile *factoringremittance.RemittanceProfile) error
}

type RemittanceLineAction string

const (
	RemittanceLineActionMatch   = RemittanceLineAction("Match")
	RemittanceLineActionUnmatch = RemittanceLineAction("Unmatch")
	RemittanceLineActionIgnore  = RemittanceLineAction("Ignore")
	RemittanceLineActionRestore = RemittanceLineAction("Restore")
	RemittanceLineActionSetKind = RemittanceLineAction("SetKind")
)

func (a RemittanceLineAction) IsValid() bool {
	switch a {
	case RemittanceLineActionMatch,
		RemittanceLineActionUnmatch,
		RemittanceLineActionIgnore,
		RemittanceLineActionRestore,
		RemittanceLineActionSetKind:
		return true
	default:
		return false
	}
}

func AllRemittanceLineActions() []RemittanceLineAction {
	return []RemittanceLineAction{
		RemittanceLineActionMatch,
		RemittanceLineActionUnmatch,
		RemittanceLineActionIgnore,
		RemittanceLineActionRestore,
		RemittanceLineActionSetKind,
	}
}

type CreateRemittanceRequest struct {
	TenantInfo         pagination.TenantInfo                  `json:"-"`
	FactoringCompanyID pulid.ID                               `json:"factoringCompanyId"`
	DefaultKind        factoringremittance.RemittanceLineKind `json:"defaultKind"`
	RemittedOn         *int64                                 `json:"remittedOn"`
}

type StageRemittanceRequest struct {
	TenantInfo  pagination.TenantInfo                  `json:"-"`
	ID          pulid.ID                               `json:"id"`
	Version     int64                                  `json:"version"`
	DocumentID  pulid.ID                               `json:"documentId"`
	Mapping     map[string]int                         `json:"mapping"`
	DefaultKind factoringremittance.RemittanceLineKind `json:"defaultKind"`
	RemittedOn  *int64                                 `json:"remittedOn"`
}

type RemittanceVersionRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
	Version    int64                 `json:"version"`
}

type UpdateRemittanceLineRequest struct {
	TenantInfo   pagination.TenantInfo                  `json:"-"`
	RemittanceID pulid.ID                               `json:"remittanceId"`
	Version      int64                                  `json:"version"`
	LineID       pulid.ID                               `json:"lineId"`
	Action       RemittanceLineAction                   `json:"action"`
	ItemID       pulid.ID                               `json:"itemId"`
	Kind         factoringremittance.RemittanceLineKind `json:"kind"`
	Reason       string                                 `json:"reason"`
}

type DiscardRemittanceRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
	Version    int64                 `json:"version"`
	Reason     string                `json:"reason"`
}

type PlannedRemittanceLine struct {
	LineID        pulid.ID                               `json:"lineId"`
	RowNumber     int                                    `json:"rowNumber"`
	Kind          factoringremittance.RemittanceLineKind `json:"kind"`
	InvoiceNumber string                                 `json:"invoiceNumber"`
	SubmissionID  pulid.ID                               `json:"submissionId"`
	ItemID        pulid.ID                               `json:"itemId"`
	CashMinor     int64                                  `json:"cashMinor"`
	Movement      *factoringledgerports.PlannedMovement  `json:"movement"`
}

type RemittancePlan struct {
	Remittance      *factoringremittance.Remittance `json:"remittance"`
	Lines           []*PlannedRemittanceLine        `json:"lines"`
	FundedCount     int                             `json:"fundedCount"`
	RejectedCount   int                             `json:"rejectedCount"`
	ReleasedCount   int                             `json:"releasedCount"`
	ChargebackCount int                             `json:"chargebackCount"`
	LeftOpenCount   int                             `json:"leftOpenCount"`
	AdvanceMinor    int64                           `json:"advanceMinor"`
	FeeMinor        int64                           `json:"feeMinor"`
	ReleasedMinor   int64                           `json:"releasedMinor"`
	ChargebackMinor int64                           `json:"chargebackMinor"`
}

type ApplyRemittanceResult struct {
	Remittance *factoringremittance.Remittance `json:"remittance"`
	Entries    []*factoringledger.ReserveEntry `json:"entries"`
}

type RemittanceEmailAttachment struct {
	DocumentID  pulid.ID `json:"documentId"`
	FileName    string   `json:"fileName"`
	ContentType string   `json:"contentType"`
}

type IngestRemittanceEmailRequest struct {
	TenantInfo       pagination.TenantInfo       `json:"-"`
	InboundMessageID pulid.ID                    `json:"inboundMessageId"`
	FromAddress      string                      `json:"fromAddress"`
	ReceivedAt       int64                       `json:"receivedAt"`
	Attachments      []RemittanceEmailAttachment `json:"attachments"`
}

type IngestRemittanceEmailResult struct {
	FactoringCompanyID pulid.ID   `json:"factoringCompanyId"`
	FactorName         string     `json:"factorName"`
	RemittanceIDs      []pulid.ID `json:"remittanceIds"`
	Problem            string     `json:"problem"`
}

type RemittanceLinePlan struct {
	Before     *factoringremittance.Remittance     `json:"before"`
	After      *factoringremittance.Remittance     `json:"after"`
	LineBefore *factoringremittance.RemittanceLine `json:"lineBefore"`
	LineAfter  *factoringremittance.RemittanceLine `json:"lineAfter"`
	Comment    string                              `json:"comment"`
}

type RemittanceDiscardPlan struct {
	Before *factoringremittance.Remittance `json:"before"`
	After  *factoringremittance.Remittance `json:"after"`
}

type StagePendingResult struct {
	Remittance *factoringremittance.Remittance `json:"remittance"`
	Waiting    bool                            `json:"waiting"`
}

type RemittanceService interface {
	Get(ctx context.Context, req GetRemittanceRequest) (*factoringremittance.Remittance, error)
	ListConnection(
		ctx context.Context,
		req *ListRemittanceConnectionRequest,
	) (*pagination.CursorListResult[*factoringremittance.Remittance], error)
	ListLineConnection(
		ctx context.Context,
		req *ListRemittanceLineConnectionRequest,
	) (*pagination.CursorListResult[*factoringremittance.RemittanceLine], error)
	Create(
		ctx context.Context,
		req *CreateRemittanceRequest,
		actor *permissionports.RequestActor,
	) (*factoringremittance.Remittance, error)
	Stage(
		ctx context.Context,
		req *StageRemittanceRequest,
		actor *permissionports.RequestActor,
	) (*factoringremittance.Remittance, error)
	StagePending(
		ctx context.Context,
		req *RemittanceVersionRequest,
		giveUp bool,
	) (*StagePendingResult, error)
	Rematch(
		ctx context.Context,
		req *RemittanceVersionRequest,
		actor *permissionports.RequestActor,
	) (*factoringremittance.Remittance, error)
	PlanUpdateLine(
		ctx context.Context,
		req *UpdateRemittanceLineRequest,
		actor *permissionports.RequestActor,
	) (*RemittanceLinePlan, error)
	UpdateLine(
		ctx context.Context,
		req *UpdateRemittanceLineRequest,
		actor *permissionports.RequestActor,
	) (*factoringremittance.Remittance, error)
	PlanApply(
		ctx context.Context,
		req *RemittanceVersionRequest,
		actor *permissionports.RequestActor,
	) (*RemittancePlan, error)
	Apply(
		ctx context.Context,
		req *RemittanceVersionRequest,
		actor *permissionports.RequestActor,
	) (*ApplyRemittanceResult, error)
	PlanDiscard(
		ctx context.Context,
		req *DiscardRemittanceRequest,
		actor *permissionports.RequestActor,
	) (*RemittanceDiscardPlan, error)
	Discard(
		ctx context.Context,
		req *DiscardRemittanceRequest,
		actor *permissionports.RequestActor,
	) (*factoringremittance.Remittance, error)
	IngestEmail(
		ctx context.Context,
		req *IngestRemittanceEmailRequest,
	) (*IngestRemittanceEmailResult, error)
}

type RemittanceStageStarter interface {
	StartStage(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		remittance *factoringremittance.Remittance,
	)
}
