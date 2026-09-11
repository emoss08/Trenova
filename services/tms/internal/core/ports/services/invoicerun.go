package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// PreviewInvoiceRunRequest builds a proposal for a period. CustomerIDs empty
// means every customer whose schedule is due.
type PreviewInvoiceRunRequest struct {
	TenantInfo  pagination.TenantInfo
	CustomerIDs []pulid.ID
	PeriodStart int64
	PeriodEnd   int64
	InvoiceDate int64
	Source      invoicerun.Source
	Cycle       customer.BillingCycle
}

// ItemExclusion pulls one shipment off the statement. The reason is required
// because next period's biller has to be able to see why.
type ItemExclusion struct {
	ItemID pulid.ID
	Reason string
}

// ItemMove sends one shipment to another group of the same customer.
type ItemMove struct {
	ItemID        pulid.ID
	TargetGroupID pulid.ID
}

// AdjustInvoiceRunMembershipRequest carries a whole operator edit, so the move
// and the exclusion land in one transaction and produce one audit entry rather
// than a trail of single-row changes nobody can read back.
type AdjustInvoiceRunMembershipRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Exclude    []ItemExclusion
	Include    []pulid.ID
	Moves      []ItemMove
}

type CommitInvoiceRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
}

// CommitGroupResult is what became of one proposed invoice.
type CommitGroupResult struct {
	GroupID       pulid.ID `json:"groupId"`
	GroupLabel    string   `json:"groupLabel"`
	Success       bool     `json:"success"`
	Skipped       bool     `json:"skipped"`
	InvoiceID     pulid.ID `json:"invoiceId,omitempty"`
	InvoiceNumber string   `json:"invoiceNumber,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// CommitInvoiceRunResult mirrors the bulk envelope the rest of the codebase
// already uses, so a partially failed run reads the same way as a partially
// failed bulk transfer.
type CommitInvoiceRunResult struct {
	Run          *invoicerun.InvoiceRun `json:"run"`
	Results      []CommitGroupResult    `json:"results"`
	TotalCount   int                    `json:"totalCount"`
	SuccessCount int                    `json:"successCount"`
	SkippedCount int                    `json:"skippedCount"`
	ErrorCount   int                    `json:"errorCount"`
}

type CancelInvoiceRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Reason     string
}

// InvoiceRunSweepResult is what one pass of the scheduled sweep did.
type InvoiceRunSweepResult struct {
	SchedulesDue    int `json:"schedulesDue"`
	RunsBuilt       int `json:"runsBuilt"`
	InvoicesCreated int `json:"invoicesCreated"`
	GroupsSkipped   int `json:"groupsSkipped"`
	Failed          int `json:"failed"`
}

// InvoiceRunSweeper is the scheduled half of statement billing.
//
// It is an interface here rather than a concrete dependency because the billing
// job package is imported by the invoice service, and the run service depends on
// the invoice service — taking the concrete type would close that loop.
type InvoiceRunSweeper interface {
	SweepDueSchedules(
		ctx context.Context,
		actor *RequestActor,
	) (*InvoiceRunSweepResult, error)
}
