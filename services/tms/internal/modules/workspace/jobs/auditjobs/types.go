package auditjobs

import (
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditintegrity"
)

type SealResult struct {
	Tenants int `json:"tenants"`
	Seals   int `json:"seals"`
	Failed  int `json:"failed"`
}

type VerifyResult struct {
	Tenants int `json:"tenants"`
	Broken  int `json:"broken"`
}

type ExportResult struct {
	Status   string `json:"status"`
	RowCount int64  `json:"rowCount"`
}

func sealResult(r *auditintegrity.SealResult) *SealResult {
	return &SealResult{Tenants: r.Tenants, Seals: r.Seals, Failed: r.Failed}
}

func verifyResult(r *auditintegrity.VerifyResult) *VerifyResult {
	return &VerifyResult{Tenants: r.Tenants, Broken: r.Broken}
}
