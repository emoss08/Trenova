package detentionjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs"
)

const SweepDetentionNoticesWorkflowName = "SweepDetentionNoticesWorkflow"

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
