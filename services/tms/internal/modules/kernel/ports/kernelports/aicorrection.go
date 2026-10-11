package kernelports

import (
	"github.com/emoss08/trenova/pkg/pagination"
)

type PurgeExpiredAICorrectionsRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
}
