package factoringsubmissionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const TransportGroup = `group:"factoring_transports"`

type PlanSubmissionRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
	InvoiceIDs         []pulid.ID            `json:"invoiceIds"`
	AgentRunID         pulid.ID              `json:"-"`
}

type SubmissionPlanItem struct {
	InvoiceID     pulid.ID          `json:"invoiceId"`
	InvoiceNumber string            `json:"invoiceNumber"`
	CustomerID    pulid.ID          `json:"customerId"`
	CustomerName  string            `json:"customerName"`
	InvoiceDate   int64             `json:"invoiceDate"`
	DueDate       *int64            `json:"dueDate"`
	Funding       factoring.Funding `json:"funding"`
}

type SubmissionPlan struct {
	FactoringCompanyID   pulid.ID                            `json:"factoringCompanyId"`
	FactoringCompanyName string                              `json:"factoringCompanyName"`
	Agreement            *factoring.Agreement                `json:"agreement"`
	TransportMethod      factoringsubmission.TransportMethod `json:"transportMethod"`
	TransportTarget      string                              `json:"transportTarget"`
	AttachPackets        bool                                `json:"attachPackets"`
	CurrencyCode         string                              `json:"currencyCode"`
	Items                []*SubmissionPlanItem               `json:"items"`
	InvoiceCount         int                                 `json:"invoiceCount"`
	FaceMinor            int64                               `json:"faceMinor"`
	ExpectedAdvanceMinor int64                               `json:"expectedAdvanceMinor"`
	ExpectedFeeMinor     int64                               `json:"expectedFeeMinor"`
	ExpectedReserveMinor int64                               `json:"expectedReserveMinor"`
	IdempotencyKey       string                              `json:"-"`
}

type SubmissionActionRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	SubmissionID pulid.ID              `json:"submissionId"`
	Version      int64                 `json:"version"`
	Reason       string                `json:"reason"`
}

type SubmissionActionPlan struct {
	Before *factoringsubmission.Submission `json:"before"`
	After  *factoringsubmission.Submission `json:"after"`
}

type DeliverSubmissionRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	SubmissionID pulid.ID              `json:"submissionId"`
	Group        int                   `json:"group"`
	Attempt      int                   `json:"attempt"`
}

type FailSubmissionRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	SubmissionID pulid.ID              `json:"submissionId"`
	Group        int                   `json:"group"`
	Attempts     int                   `json:"attempts"`
	Cause        string                `json:"cause"`
}

type DeliverSubmissionResult struct {
	Status    factoringsubmission.SubmissionStatus `json:"status"`
	Reference string                               `json:"reference"`
	Parts     int                                  `json:"parts"`
	Skipped   bool                                 `json:"skipped"`
}

type SubmissionService interface {
	Plan(ctx context.Context, req *PlanSubmissionRequest) (*SubmissionPlan, error)
	Submit(
		ctx context.Context,
		req *PlanSubmissionRequest,
		actor *permissionports.RequestActor,
	) (*factoringsubmission.Submission, error)
	PlanRetry(ctx context.Context, req *SubmissionActionRequest) (*SubmissionActionPlan, error)
	Retry(
		ctx context.Context,
		req *SubmissionActionRequest,
		actor *permissionports.RequestActor,
	) (*factoringsubmission.Submission, error)
	PlanCancel(ctx context.Context, req *SubmissionActionRequest) (*SubmissionActionPlan, error)
	Cancel(
		ctx context.Context,
		req *SubmissionActionRequest,
		actor *permissionports.RequestActor,
	) (*factoringsubmission.Submission, error)
	Get(ctx context.Context, req GetSubmissionRequest) (*factoringsubmission.Submission, error)
	List(
		ctx context.Context,
		req *ListSubmissionConnectionRequest,
	) (*pagination.CursorListResult[*factoringsubmission.Submission], error)
	ListItems(
		ctx context.Context,
		req *ListItemsBySubmissionIDsRequest,
	) (map[pulid.ID][]*factoringsubmission.SubmissionItem, error)
	LiveItems(ctx context.Context, req *ListLiveItemsRequest) (map[pulid.ID]*LiveItem, error)
	PrepareDocuments(ctx context.Context, req *DeliverSubmissionRequest) error
	Deliver(
		ctx context.Context,
		req *DeliverSubmissionRequest,
	) (*DeliverSubmissionResult, error)
	MarkFailed(ctx context.Context, req *FailSubmissionRequest) error
}

type TransportInput struct {
	Method           factoringsubmission.TransportMethod `json:"method"`
	EmailRecipients  []string                            `json:"emailRecipients"`
	AttachPackets    bool                                `json:"attachPackets"`
	SftpHost         string                              `json:"sftpHost"`
	SftpPort         int                                 `json:"sftpPort"`
	SftpUsername     string                              `json:"sftpUsername"`
	SftpAuthMethod   factoringsubmission.SftpAuthMethod  `json:"sftpAuthMethod"`
	SftpSecret       *string                             `json:"-"`
	SftpKnownHostKey string                              `json:"sftpKnownHostKey"`
	SftpDirectory    string                              `json:"sftpDirectory"`
}

type SaveTransportRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
	Version            *int64                `json:"version"`
	Input              TransportInput        `json:"input"`
}

type TestTransportRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
}

type TransportTestResult struct {
	Transport *factoringsubmission.Transport `json:"transport"`
	Passed    bool                           `json:"passed"`
	Message   string                         `json:"message"`
}

type TransportService interface {
	Get(ctx context.Context, req GetTransportRequest) (*factoringsubmission.Transport, error)
	Save(
		ctx context.Context,
		req *SaveTransportRequest,
		actor *permissionports.RequestActor,
	) (*factoringsubmission.Transport, error)
	Test(
		ctx context.Context,
		req *TestTransportRequest,
		actor *permissionports.RequestActor,
	) (*TransportTestResult, error)
}

type SubmissionSendStarter interface {
	StartSend(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		submission *factoringsubmission.Submission,
	)
}
