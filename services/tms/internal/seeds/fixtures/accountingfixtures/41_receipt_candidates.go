package accountingfixtures

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem/bankreceiptworkitemdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const unmatchedReason = "No unique customer payment match found for bank receipt"

type seedCandidateReceipt struct {
	reference string
	memo      string
	invoice   string
	delta     int64
	amount    int64
	ageDays   int64
}

var seedCandidateReceipts = []seedCandidateReceipt{
	{
		reference: "SEED-INV-0006",
		memo:      "ACH CREDIT REMITTANCE",
		invoice:   "SEED-INV-0006",
		ageDays:   1,
	},
	{
		reference: "SEED-INV-0003",
		memo:      "LOCKBOX DEPOSIT CHK 40877",
		invoice:   "SEED-INV-0003",
		delta:     -12_000,
		ageDays:   2,
	},
	{
		reference: "SEED-INV-0008",
		memo:      "ACH CREDIT REMITTANCE",
		invoice:   "SEED-INV-0008",
		delta:     4_510,
		ageDays:   2,
	},
	{
		reference: "LOCKBOX 2207",
		memo:      "LOCKBOX DEPOSIT REMIT SEED-INV-0010",
		invoice:   "SEED-INV-0010",
		ageDays:   3,
	},
	{
		reference: "WIRE 0915",
		memo:      "INCOMING WIRE - NO REMITTANCE DETAIL",
		amount:    318_844,
		ageDays:   3,
	},
}

type ReceiptCandidatesSeed struct{}

func (s *ReceiptCandidatesSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := tenantfixtures.UserByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}
	now := timeutils.NowUnix()
	for idx := range seedCandidateReceipts {
		spec := &seedCandidateReceipts[idx]
		if err = s.ensure(ctx, db, &candidateSeedRefs{
			orgID: org.ID, buID: org.BusinessUnitID, adminID: admin.ID, now: now,
		}, spec); err != nil {
			return fmt.Errorf("bank receipt %s: %w", spec.reference, err)
		}
	}

	return nil
}

type candidateSeedRefs struct {
	orgID   pulid.ID
	buID    pulid.ID
	adminID pulid.ID
	now     int64
}

func (s *ReceiptCandidatesSeed) ensure(
	ctx context.Context,
	db *keel.DB,
	refs *candidateSeedRefs,
	spec *seedCandidateReceipt,
) error {
	br := bankreceiptdb.BankReceipts
	exists, err := br.AllTenants().Where(
		br.OrganizationID.Eq(refs.orgID),
		br.BusinessUnitID.Eq(refs.buID),
		br.ReferenceNumber.Eq(spec.reference),
		br.Memo.Eq(spec.memo),
	).Exists(ctx, db)
	if err != nil || exists {
		return err
	}
	amount := spec.amount
	if spec.invoice != "" {
		open, openErr := s.openBalance(ctx, db, refs, spec.invoice)
		if openErr != nil || open <= 0 {
			return openErr
		}
		amount = open + spec.delta
	}
	receipt := &bankreceipt.BankReceipt{
		ID:              pulid.MustNew("brcpt_"),
		OrganizationID:  refs.orgID,
		BusinessUnitID:  refs.buID,
		ReceiptDate:     refs.now - spec.ageDays*receivableSeedDay,
		AmountMinor:     amount,
		ReferenceNumber: spec.reference,
		Memo:            spec.memo,
		Status:          bankreceipt.StatusException,
		ExceptionReason: unmatchedReason,
		CreatedByID:     refs.adminID,
		UpdatedByID:     refs.adminID,
	}
	if err = br.Create(ctx, db, receipt); err != nil {
		return fmt.Errorf("insert bank receipt: %w", err)
	}
	if err = bankreceiptworkitemdb.WorkItems.Create(ctx, db, &bankreceiptworkitem.WorkItem{
		ID:             pulid.MustNew("brwi_"),
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		BankReceiptID:  receipt.ID,
		Status:         bankreceiptworkitem.StatusOpen,
		CreatedByID:    refs.adminID,
		UpdatedByID:    refs.adminID,
	}); err != nil {
		return fmt.Errorf("insert bank receipt work item: %w", err)
	}

	return nil
}

func (s *ReceiptCandidatesSeed) openBalance(
	ctx context.Context,
	db *keel.DB,
	refs *candidateSeedRefs,
	number string,
) (int64, error) {
	inv := invoicedb.Invoices
	found, err := inv.AllTenants().Where(
		inv.OrganizationID.Eq(refs.orgID),
		inv.BusinessUnitID.Eq(refs.buID),
		inv.Number.Eq(number),
		inv.Status.Eq(invoice.StatusPosted),
	).First(ctx, db)
	if errors.Is(err, keel.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find invoice %s: %w", number, err)
	}

	return found.OpenBalanceMinor(), nil
}
