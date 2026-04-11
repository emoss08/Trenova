package seqgen

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type SequenceRequest struct {
	Type  tenant.SequenceType
	OrgID pulid.ID
	BuID  pulid.ID
	Year  int
	Month int
	Count int
}

type LastGeneratedRequest struct {
	Type  tenant.SequenceType
	OrgID pulid.ID
	BuID  pulid.ID
	Year  int
	Month int
	Value string
}

type GenerateRequest struct {
	Type             tenant.SequenceType
	OrgID            pulid.ID
	BuID             pulid.ID
	Count            int
	Time             time.Time
	Format           *tenant.SequenceFormat
	LocationCode     string
	BusinessUnitCode string
}

type SequenceStore interface {
	GetNextSequence(ctx context.Context, req *SequenceRequest) (int64, error)
	GetNextSequenceBatch(ctx context.Context, req *SequenceRequest) ([]int64, error)
	UpdateLastGenerated(ctx context.Context, req *LastGeneratedRequest) error
}

type FormatProvider interface {
	GetFormat(
		ctx context.Context,
		sequenceType tenant.SequenceType,
		orgID, buID pulid.ID,
	) (*tenant.SequenceFormat, error)
}

type Generator interface {
	Generate(ctx context.Context, req *GenerateRequest) (string, error)
	GenerateBatch(ctx context.Context, req *GenerateRequest) ([]string, error)
	GenerateShipmentProNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
	GenerateConsolidationNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
	GenerateInvoiceNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
	GenerateCreditMemoNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
	GenerateDebitMemoNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
	GenerateWorkOrderNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
	GenerateJournalBatchNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
	GenerateJournalEntryNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
	GenerateManualJournalRequestNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
}
