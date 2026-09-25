package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// AIAuditExportWorkflowName writes a large trail export on the report
	// queue. It is started by name, so the service that starts it does not
	// import the jobs package that runs it.
	AIAuditExportWorkflowName = "AIAuditExportWorkflow"
	// AIAuditVerifyWorkflowName checks one tenant's chain, or every tenant's
	// when no tenant is named.
	AIAuditVerifyWorkflowName = "VerifyAIAuditChainWorkflow"

	AIAuditExportReadyEvent   = "ai_audit_export_ready"
	AIAuditExportFailedEvent  = "ai_audit_export_failed"
	AIAuditChainMismatchEvent = "ai_audit_chain_mismatch"
	AIAuditExportResource     = "ai-audit-export"
	AIAuditChainResource      = "ai-audit-chain"
	AIAuditExportRecordEntity = "ai_audit_export"
)

// AIAuditExportPayload names the export a workflow writes.
type AIAuditExportPayload struct {
	ExportID       pulid.ID `json:"exportId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

func (p *AIAuditExportPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

// AIAuditVerifyPayload names the tenant a verification checks. Both empty
// checks every tenant with a trail.
type AIAuditVerifyPayload struct {
	OrganizationID pulid.ID `json:"organizationId,omitempty"`
	BusinessUnitID pulid.ID `json:"businessUnitId,omitempty"`
	RequestedBy    pulid.ID `json:"requestedBy,omitempty"`
}

// FieldCeilings answers the most sensitive tier a reader may see on a
// resource, remembered for the length of one request.
type FieldCeilings interface {
	For(ctx context.Context, resource permission.Resource) permission.FieldSensitivity
}

// ListAIAuditEventsRequest reads the trail. A "personId" field filter is
// rewritten into "acted for or decided by this person" before the query.
type ListAIAuditEventsRequest struct {
	Filter  *pagination.QueryOptions
	Cursor  pagination.CursorInfo
	Columns []string
}

// AIAuditChainStatus is a tenant's chain as a reader sees it.
type AIAuditChainStatus struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	// Signed is false when no chain key is configured and new rows are
	// linked with plain SHA-256.
	Signed                 bool                       `json:"signed"`
	ActiveKeyID            string                     `json:"activeKeyId"`
	FirstSeq               int64                      `json:"firstSeq"`
	LastSeq                int64                      `json:"lastSeq"`
	LastHash               string                     `json:"lastHash"`
	SealedThroughSeq       int64                      `json:"sealedThroughSeq"`
	LastVerifiedSeq        int64                      `json:"lastVerifiedSeq"`
	LastVerifiedAt         *int64                     `json:"lastVerifiedAt"`
	LastVerificationStatus aiaudit.VerificationStatus `json:"lastVerificationStatus"`
	FailedSeq              *int64                     `json:"failedSeq"`
	Detail                 string                     `json:"detail"`
	// Verifying is set when a verification was just started for this tenant.
	Verifying bool `json:"verifying"`
}

// RequestAIAuditExportRequest asks for the trail in a range as a file.
type RequestAIAuditExportRequest struct {
	Actor  *RequestActor
	Format aiaudit.ExportFormat
	From   int64
	To     int64
	Filter *aiaudit.ExportFilter
}

// AIAuditExportDownload is a short-lived link to an export's file.
type AIAuditExportDownload struct {
	URL       string `json:"url"`
	FileName  string `json:"fileName"`
	ExpiresAt int64  `json:"expiresAt"`
	SHA256    string `json:"sha256"`
}

type GetAIAuditExportDownloadRequest struct {
	Actor    *RequestActor
	ExportID pulid.ID
}

// AIAuditService is the trail as the API reads and exports it.
type AIAuditService interface {
	ListEvents(
		ctx context.Context,
		req *ListAIAuditEventsRequest,
	) (*pagination.CursorListResult[*aiaudit.AIAuditEvent], error)
	GetEvent(
		ctx context.Context,
		req repositories.GetAIAuditEventRequest,
	) (*aiaudit.AIAuditEvent, error)
	// ReaderArguments are an event's recorded arguments with every path above
	// the reader's ceiling on the tool's resource withheld.
	ReaderArguments(
		ctx context.Context,
		event *aiaudit.AIAuditEvent,
		ceilings FieldCeilings,
	) map[string]any
	ChainStatus(ctx context.Context, tenantInfo pagination.TenantInfo) (*AIAuditChainStatus, error)
	RequestVerification(ctx context.Context, actor *RequestActor) (*AIAuditChainStatus, error)
	RequestExport(
		ctx context.Context,
		req *RequestAIAuditExportRequest,
	) (*aiaudit.AIAuditExport, error)
	GetExport(
		ctx context.Context,
		req repositories.GetAIAuditExportRequest,
	) (*aiaudit.AIAuditExport, error)
	ListExports(
		ctx context.Context,
		req *repositories.ListAIAuditExportsRequest,
	) (*pagination.CursorListResult[*aiaudit.AIAuditExport], error)
	ExportDownload(
		ctx context.Context,
		req *GetAIAuditExportDownloadRequest,
	) (*AIAuditExportDownload, error)
	// TraceURL is where a trace opens in the tracing backend, or empty when
	// none is configured.
	TraceURL(traceID string) string
	// SourcePruneHorizon is the newest timestamp a source's own retention
	// sweep may delete before: rows after it may not be in the trail yet.
	SourcePruneHorizon(ctx context.Context, source aiaudit.Source) (int64, error)
}
