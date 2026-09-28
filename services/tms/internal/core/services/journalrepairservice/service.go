package journalrepairservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/invoiceledger"
	"github.com/emoss08/trenova/internal/core/services/journalposting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const (
	KindCreditMemo       = "CreditMemo"
	KindDriverSettlement = "DriverSettlement"
)

type Deps struct {
	DB       ports.DBConnection
	Repo     repositories.JournalRepairRepository
	Controls invoiceledger.ControlReader
	Periods  journalposting.PeriodReader
	Journals repositories.JournalPostingRepository
	Ledger   invoiceledger.LedgerAppender
	Numbers  journalposting.Numberer
	Policy   invoiceledger.Policy
	Now      func() int64
}

type Service struct {
	deps Deps
}

type Request struct {
	OrganizationID pulid.ID
	ActorID        pulid.ID
	DryRun         bool
	BatchSize      int
}

type Skip struct {
	Kind           string
	ID             pulid.ID
	Number         string
	OrganizationID pulid.ID
	Reason         string
}

type Report struct {
	CreditMemosJournaled  int
	CreditMemosLedgerOnly int
	PaymentsJournaled     int
	PaymentsLinked        int
	Skipped               []Skip
}

func New(deps *Deps) *Service {
	return &Service{deps: *deps}
}

func (s *Service) Repair(ctx context.Context, req *Request) (*Report, error) {
	if req.ActorID.IsNil() {
		return nil, errortypes.NewValidationError(
			"actorId",
			errortypes.ErrRequired,
			"The repair needs a user to record as the author of the journals",
		)
	}

	report := &Report{}
	controls := make(map[pulid.ID]*tenant.AccountingControl)
	if err := s.repairCreditMemos(ctx, req, report); err != nil {
		return nil, err
	}
	if err := s.repairDriverPayments(ctx, req, report, controls); err != nil {
		return nil, err
	}
	return report, nil
}

func (s *Service) writer() journalposting.Writer {
	return journalposting.Writer{
		Periods:  s.deps.Periods,
		Numbers:  s.deps.Numbers,
		Journals: s.deps.Journals,
	}
}

func (s *Service) poster() *invoiceledger.Poster {
	return &invoiceledger.Poster{
		Controls: s.deps.Controls,
		Policy:   s.deps.Policy,
		Writer:   s.writer(),
		Ledger:   s.deps.Ledger,
	}
}

func (s *Service) repairCreditMemos(ctx context.Context, req *Request, report *Report) error {
	after := pulid.Nil
	for {
		batch, err := s.deps.Repo.ListUnjournaledAdjustmentMemos(
			ctx,
			&repositories.ListJournalRepairRequest{
				OrganizationID: req.OrganizationID,
				AfterID:        after,
				Limit:          req.BatchSize,
			},
		)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}

		for _, candidate := range batch {
			after = candidate.Memo.ID
			if candidate.SourceMissing {
				report.Skipped = append(report.Skipped, Skip{
					Kind:           KindCreditMemo,
					ID:             candidate.Memo.ID,
					Number:         candidate.Memo.Number,
					OrganizationID: candidate.Memo.OrganizationID,
					Reason:         "the invoice adjustment that issued it is missing or belongs to another organization or business unit",
				})
				continue
			}
			ledgerReq := invoiceledger.CreditMemoRequest(
				candidate.Kind,
				candidate.Memo,
				candidate.SourceInvoiceID,
				req.ActorID,
			)
			ledgerReq.Now = s.deps.Now()
			ledgerReq.LedgerOnly = ledgerReq.LedgerOnly || candidate.Journaled

			err = s.inTx(ctx, req.DryRun, func(txCtx context.Context) error {
				if req.DryRun {
					_, planErr := s.poster().Plan(txCtx, ledgerReq)
					return planErr
				}
				_, postErr := s.poster().Post(txCtx, ledgerReq)
				return postErr
			})
			if err != nil {
				if err = report.skip(err, Skip{
					Kind:           KindCreditMemo,
					ID:             candidate.Memo.ID,
					Number:         candidate.Memo.Number,
					OrganizationID: candidate.Memo.OrganizationID,
				}); err != nil {
					return err
				}
				continue
			}
			if ledgerReq.LedgerOnly {
				report.CreditMemosLedgerOnly++
			} else {
				report.CreditMemosJournaled++
			}
		}
	}
}

func (s *Service) repairDriverPayments(
	ctx context.Context,
	req *Request,
	report *Report,
	controls map[pulid.ID]*tenant.AccountingControl,
) error {
	after := pulid.Nil
	for {
		batch, err := s.deps.Repo.ListUnjournaledDriverPayments(
			ctx,
			&repositories.ListJournalRepairRequest{
				OrganizationID: req.OrganizationID,
				AfterID:        after,
				Limit:          req.BatchSize,
			},
		)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}

		for _, candidate := range batch {
			settlement := candidate.Settlement
			after = settlement.ID
			err = s.inTx(ctx, req.DryRun, func(txCtx context.Context) error {
				return s.repairDriverPayment(txCtx, req, candidate, controls)
			})
			if err != nil {
				if err = report.skip(err, Skip{
					Kind:           KindDriverSettlement,
					ID:             settlement.ID,
					Number:         settlement.SettlementNumber,
					OrganizationID: settlement.OrganizationID,
				}); err != nil {
					return err
				}
				continue
			}
			if candidate.JournalBatchID.IsNotNil() {
				report.PaymentsLinked++
			} else {
				report.PaymentsJournaled++
			}
		}
	}
}

func (s *Service) repairDriverPayment(
	ctx context.Context,
	req *Request,
	candidate *repositories.DriverPaymentRepair,
	controls map[pulid.ID]*tenant.AccountingControl,
) error {
	settlement := candidate.Settlement
	if candidate.JournalBatchID.IsNotNil() {
		if req.DryRun {
			return nil
		}
		return s.stampPaidBatch(ctx, settlement, candidate.JournalBatchID)
	}

	control, err := s.control(ctx, settlement.OrganizationID, controls)
	if err != nil {
		return err
	}
	journal, err := driversettlementservice.PaymentJournal(
		settlement,
		control,
		req.ActorID,
		*settlement.PaidAt,
		s.deps.Now(),
	)
	if err != nil {
		return err
	}
	if journal == nil {
		return errNothingToRepair
	}

	if req.DryRun {
		_, err = s.writer().Plan(ctx, journal.WriteRequest())
		return err
	}
	result, err := s.writer().Write(ctx, journal.WriteRequest())
	if err != nil {
		return err
	}
	return s.stampPaidBatch(ctx, settlement, result.BatchID)
}

func (s *Service) stampPaidBatch(
	ctx context.Context,
	settlement *driversettlement.Settlement,
	batchID pulid.ID,
) error {
	return s.deps.Repo.SetDriverSettlementPaidBatch(
		ctx,
		&repositories.SetDriverSettlementPaidBatchParams{
			TenantInfo: pagination.TenantInfo{
				OrgID: settlement.OrganizationID,
				BuID:  settlement.BusinessUnitID,
			},
			SettlementID: settlement.ID,
			BatchID:      batchID,
		},
	)
}

func (s *Service) control(
	ctx context.Context,
	orgID pulid.ID,
	controls map[pulid.ID]*tenant.AccountingControl,
) (*tenant.AccountingControl, error) {
	if control, ok := controls[orgID]; ok {
		return control, nil
	}
	control, err := s.deps.Controls.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	controls[orgID] = control
	return control, nil
}

var errNothingToRepair = errors.New("nothing to repair")

func (r *Report) skip(err error, skip Skip) error { //nolint:gocritic // Skip is appended by value
	switch {
	case errors.Is(err, invoiceledger.ErrNoLedgerEntry):
		skip.Reason = "revenue is not recognized when an invoice posts, so it books no ledger entry"
	case errors.Is(err, errNothingToRepair):
		skip.Reason = "no payable was booked for it, so no payment journal is needed"
	case errortypes.IsBusinessError(err), errortypes.IsError(err), errortypes.IsNotFoundError(err),
		errortypes.IsConflictError(err), errortypes.IsMultiError(err):
		skip.Reason = err.Error()
	default:
		return fmt.Errorf("repair %s %s: %w", skip.Kind, skip.ID, err)
	}
	r.Skipped = append(r.Skipped, skip)
	return nil
}

func (s *Service) inTx(ctx context.Context, dryRun bool, fn func(context.Context) error) error {
	if dryRun {
		return fn(ctx)
	}
	return s.deps.DB.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		return fn(txCtx)
	})
}
