package factoringports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringwatchtowerports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetOverviewRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
	At                 int64                 `json:"at"`
	ReadyBefore        int64                 `json:"readyBefore"`
	ChargebackBefore   int64                 `json:"chargebackBefore"`
	MovedSince         int64                 `json:"movedSince"`
	PaceLimit          int                   `json:"paceLimit"`
	AttentionLimit     int                   `json:"attentionLimit"`
}

type OverviewFactor struct {
	Name     string             `json:"name"`
	Status   domaintypes.Status `json:"status"`
	Timezone string             `json:"timezone"`
}

type AwaitingSubmission struct {
	SubmissionID pulid.ID `json:"submissionId"`
	Number       string   `json:"number"`
	InvoiceCount int      `json:"invoiceCount"`
	FaceMinor    int64    `json:"faceMinor"`
	CurrencyCode string   `json:"currencyCode"`
	SentAt       int64    `json:"sentAt"`
	DueAt        int64    `json:"dueAt"`
}

type FundedSubmission struct {
	SubmissionID pulid.ID `json:"submissionId"`
	Number       string   `json:"number"`
	SentAt       int64    `json:"sentAt"`
	FundedAt     int64    `json:"fundedAt"`
}

type OpenRemittance struct {
	RemittanceID   pulid.ID                             `json:"remittanceId"`
	FileName       string                               `json:"fileName"`
	Status         factoringremittance.RemittanceStatus `json:"status"`
	RemittedOn     *int64                               `json:"remittedOn"`
	CreatedAt      int64                                `json:"createdAt"`
	LineCount      int                                  `json:"lineCount"`
	MatchedCount   int                                  `json:"matchedCount"`
	ExceptionCount int                                  `json:"exceptionCount"`
	TotalCashMinor int64                                `json:"totalCashMinor"`
	CurrencyCode   string                               `json:"currencyCode"`
}

type NoticeCounts struct {
	Assigned     int `json:"assigned"`
	NeedsNotice  int `json:"needsNotice"`
	Awaiting     int `json:"awaiting"`
	Acknowledged int `json:"acknowledged"`
}

type OverviewRead struct {
	Factor            OverviewFactor                                 `json:"factor"`
	Summary           *Summary                                       `json:"summary"`
	Released          VerdictTotals                                  `json:"released"`
	ChargedBack       VerdictTotals                                  `json:"chargedBack"`
	SoonestChargeback *factoringwatchtowerports.ChargebackDue        `json:"soonestChargeback"`
	Awaiting          []*AwaitingSubmission                          `json:"awaiting"`
	AwaitingCount     int                                            `json:"awaitingCount"`
	Overdue           []*factoringwatchtowerports.UnfundedSubmission `json:"overdue"`
	Failed            []*factoringwatchtowerports.FailedSubmission   `json:"failed"`
	Funded            []*FundedSubmission                            `json:"funded"`
	Remittances       []*OpenRemittance                              `json:"remittances"`
	RemittanceCount   int                                            `json:"remittanceCount"`
	Notices           NoticeCounts                                   `json:"notices"`
	Transport         *factoringsubmission.Transport                 `json:"transport"`
	Agreement         *factoring.Agreement                           `json:"agreement"`
}

type OverviewRepository interface {
	Get(ctx context.Context, req *GetOverviewRequest) (*OverviewRead, error)
	AgreementActivity(
		ctx context.Context,
		req *GetAgreementActivityRequest,
	) (*AgreementActivity, error)
}

type GetAgreementActivityRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	AgreementID pulid.ID              `json:"agreementId"`
}

type AgreementActivity struct {
	Schedules            int    `json:"schedules"`
	FundedSchedules      int    `json:"fundedSchedules"`
	Invoices             int    `json:"invoices"`
	FaceMinor            int64  `json:"faceMinor"`
	ExpectedAdvanceMinor int64  `json:"expectedAdvanceMinor"`
	ExpectedFeeMinor     int64  `json:"expectedFeeMinor"`
	FirstSentAt          *int64 `json:"firstSentAt"`
	LastSentAt           *int64 `json:"lastSentAt"`
}
