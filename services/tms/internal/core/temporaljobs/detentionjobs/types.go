package detentionjobs

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const SweepDetentionNoticesWorkflowName = "SweepDetentionNoticesWorkflow"

const StoreNoticePDFWorkflowName = "StoreDetentionNoticePDFWorkflow"

// StoreNoticePDFPayload files an already-uploaded notice PDF as a document.
//
// The bytes are not here: the send path streamed them into the upload session
// before starting this workflow, so only the session id travels. Provenance does
// travel, because it is what the generated-documents row records and re-deriving
// it would mean resolving the template a second time — by which point an
// administrator may have edited it.
type StoreNoticePDFPayload struct {
	temporaltype.BasePayload

	NoticeID     pulid.ID `json:"noticeId"`
	OccurrenceID pulid.ID `json:"occurrenceId"`
	SessionID    pulid.ID `json:"sessionId"`
	FileName     string   `json:"fileName"`
	FileSize     int64    `json:"fileSize"`

	Provenance *services.RenderedDocument `json:"provenance"`

	PrincipalType services.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID               `json:"principalId"`
	APIKeyID      pulid.ID               `json:"apiKeyId"`
}

type StoreNoticePDFResult struct {
	NoticeID    pulid.ID  `json:"noticeId"`
	DocumentID  *pulid.ID `json:"documentId"`
	CompletedAt int64     `json:"completedAt"`
}

type ListDetentionTenantsPayload struct {
	Limit int `json:"limit"`
}

type ListDetentionTenantsResult struct {
	Tenants []temporaljobs.TenantWorkItem `json:"tenants"`
}

type SweepTenantNoticesPayload struct {
	temporaljobs.TenantWorkItem
}

type SweepTenantNoticesResult struct {
	Due     int `json:"due"`
	Sent    int `json:"sent"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

type SweepDetentionNoticesResult struct {
	temporaljobs.TenantRunResult
	NoticesSent    int `json:"noticesSent"`
	NoticesSkipped int `json:"noticesSkipped"`
	NoticesFailed  int `json:"noticesFailed"`
}
