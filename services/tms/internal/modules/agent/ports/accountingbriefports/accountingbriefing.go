package accountingbriefports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AccountingBriefingSegment struct {
	Text       string                `json:"text"`
	Link       *accountingbrief.Link `json:"link,omitempty"`
	CustomerID *pulid.ID             `json:"customerId,omitempty"`
}

type AccountingBriefingAction struct {
	Link         accountingbrief.Link `json:"link"`
	Count        int                  `json:"count"`
	AmountMinor  int64                `json:"amountMinor"`
	CustomerID   *pulid.ID            `json:"customerId,omitempty"`
	CustomerName string               `json:"customerName,omitempty"`
}

type AccountingBriefing struct {
	Segments    []AccountingBriefingSegment `json:"segments"`
	Actions     []AccountingBriefingAction  `json:"actions"`
	Narrated    bool                        `json:"narrated"`
	GeneratedAt int64                       `json:"generatedAt"`
	Generation  int                         `json:"generation"`
}

type AccountingBriefingReader interface {
	Briefing(ctx context.Context, tenantInfo pagination.TenantInfo) (*AccountingBriefing, error)
}

type WriteAccountingBriefRequest struct {
	TenantInfo pagination.TenantInfo
	Trigger    accountingbrief.Trigger
}

type AccountingBriefWriter interface {
	WriteBrief(
		ctx context.Context,
		req *WriteAccountingBriefRequest,
	) (*accountingbrief.Brief, error)
}
