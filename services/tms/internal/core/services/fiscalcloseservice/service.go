// Package fiscalcloseservice turns closing a fiscal year into real accounting:
// it empties the income statement into retained earnings and carries every
// balance-sheet account forward into the first period of the next year.
package fiscalcloseservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	FiscalYearRepo   repositories.FiscalYearRepository
	FiscalPeriodRepo repositories.FiscalPeriodRepository
	GLBalanceRepo    repositories.GLBalanceRepository
	GLAccountRepo    repositories.GLAccountRepository
	AccountingRepo   repositories.AccountingControlRepository
	CustomerLedger   repositories.CustomerLedgerProjectionRepository
	JournalPostRepo  repositories.JournalPostingRepository
	JournalEntryRepo repositories.JournalEntryRepository
	SourceRepo       repositories.JournalSourceRepository
	Generator        seqgen.Generator
}

type Service struct {
	l                  *zap.Logger
	fiscalYearRepo     repositories.FiscalYearRepository
	fiscalPeriodRepo   repositories.FiscalPeriodRepository
	glBalanceRepo      repositories.GLBalanceRepository
	glAccountRepo      repositories.GLAccountRepository
	accountingRepo     repositories.AccountingControlRepository
	customerLedgerRepo repositories.CustomerLedgerProjectionRepository
	journalPostRepo    repositories.JournalPostingRepository
	journalEntryRepo   repositories.JournalEntryRepository
	sourceRepo         repositories.JournalSourceRepository
	generator          seqgen.Generator
}

func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.fiscal-close"),
		fiscalYearRepo:     p.FiscalYearRepo,
		fiscalPeriodRepo:   p.FiscalPeriodRepo,
		glBalanceRepo:      p.GLBalanceRepo,
		glAccountRepo:      p.GLAccountRepo,
		accountingRepo:     p.AccountingRepo,
		customerLedgerRepo: p.CustomerLedger,
		journalPostRepo:    p.JournalPostRepo,
		journalEntryRepo:   p.JournalEntryRepo,
		sourceRepo:         p.SourceRepo,
		generator:          p.Generator,
	}
}

// BuildPlan computes, without writing anything, the closing and opening entries
// that closing the given year would post. Anything that stops the close comes
// back as a blocker on the plan rather than as an error, so the preview endpoint
// and the close itself share one code path.
func (s *Service) BuildPlan(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
) (*fiscalclose.Plan, error) {
	plan, _, err := s.prepare(ctx, fy)

	return plan, err
}

// prepare gathers every input the plan needs and returns the resolved targets
// alongside it, because posting needs the period it may still have to create.
func (s *Service) prepare(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
) (*fiscalclose.Plan, planInputs, error) {
	in := planInputs{
		fiscalYear: fy,
		blockers:   make([]*fiscalclose.Blocker, 0),
	}
	tenantInfo := tenantOf(fy)

	control, err := s.loadAccountingControl(ctx, fy)
	if err != nil {
		return nil, in, err
	}

	retained, blocker := s.resolveRetainedEarnings(ctx, fy, control)
	in.retainedEarnings = retained
	in.blockers = appendBlocker(in.blockers, blocker)

	closingTarget, blocker, err := s.resolveClosingPeriod(ctx, fy)
	if err != nil {
		return nil, in, err
	}
	in.closingPeriod = closingTarget
	in.blockers = appendBlocker(in.blockers, blocker)

	nextYear, openingTarget, nextBlockers, err := s.resolveCarryforwardTarget(ctx, fy)
	if err != nil {
		return nil, in, err
	}
	in.nextFiscalYear = nextYear
	in.openingPeriod = openingTarget
	in.blockers = append(in.blockers, nextBlockers...)

	in.balances, err = s.glBalanceRepo.ListYearToDateBalances(
		ctx,
		repositories.ListYearToDateBalancesRequest{
			TenantInfo:   tenantInfo,
			FiscalYearID: fy.ID,
		},
	)
	if err != nil {
		return nil, in, err
	}

	in.revision, err = s.closeRevision(ctx, fy)
	if err != nil {
		return nil, in, err
	}

	in.subledgerChecks, err = s.reconcileSubledgers(ctx, fy, control, in.balances)
	if err != nil {
		return nil, in, err
	}

	return buildPlan(&in), in, nil
}

// loadAccountingControl returns the organization's accounting control, or nil
// when it has none. A tenant without one cannot close, and the blocker for that
// is raised where retained earnings is resolved.
func (s *Service) loadAccountingControl(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
) (*tenant.AccountingControl, error) {
	control, err := s.accountingRepo.GetByOrgID(ctx, fy.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return control, nil
}

// resolveRetainedEarnings loads the equity account the year result rolls into.
// The account is configured on the organization's accounting control and has to
// be a real, active equity account: booking a year of profit anywhere else
// silently corrupts the balance sheet.
func (s *Service) resolveRetainedEarnings(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
	control *tenant.AccountingControl,
) (retainedEarningsAccount, *fiscalclose.Blocker) {
	if control == nil {
		return retainedEarningsAccount{}, accountingBlocker(
			"accountingControl",
			"This organization has no accounting control record, so the year cannot be closed.",
		)
	}

	if control.DefaultRetainedEarningsAccountID.IsNil() {
		return retainedEarningsAccount{}, accountingBlocker(
			"defaultRetainedEarningsAccountId",
			"Set a default retained earnings account in Accounting Control before closing a fiscal year.",
		)
	}

	account, err := s.glAccountRepo.GetByID(ctx, repositories.GetGLAccountByIDRequest{
		ID:         control.DefaultRetainedEarningsAccountID,
		TenantInfo: tenantOf(fy),
	})
	if err != nil {
		return retainedEarningsAccount{}, accountingBlocker(
			"defaultRetainedEarningsAccountId",
			"The configured retained earnings account could not be loaded.",
		)
	}

	resolved := retainedEarningsAccount{
		ID:   account.ID,
		Code: account.AccountCode,
		Name: account.Name,
	}
	if account.AccountType != nil {
		resolved.Category = account.AccountType.Category
	}

	if resolved.Category != accounttype.CategoryEquity {
		return resolved, accountingBlocker(
			"defaultRetainedEarningsAccountId",
			fmt.Sprintf(
				"Retained earnings must point at an Equity account; %s is categorized as %s.",
				account.AccountCode,
				resolved.Category,
			),
		)
	}

	return resolved, nil
}

// resolveClosingPeriod picks the period the closing entry posts into. Closing
// entries belong in a year-end adjusting period rather than in December, so that
// the operating period keeps showing operating results; the period is created by
// the close when the carrier has not made one.
func (s *Service) resolveClosingPeriod(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
) (periodTarget, *fiscalclose.Blocker, error) {
	periods, err := s.fiscalPeriodRepo.ListByFiscalYearID(
		ctx,
		repositories.ListByFiscalYearIDRequest{
			FiscalYearID: fy.ID,
			OrgID:        fy.OrganizationID,
			BuID:         fy.BusinessUnitID,
		},
	)
	if err != nil {
		return periodTarget{}, nil, err
	}

	if len(periods) == 0 {
		return periodTarget{}, accountingBlocker(
			"periods",
			"This fiscal year has no periods, so there is nowhere to post the closing entry.",
		), nil
	}

	var lastOperating, adjusting *fiscalperiod.FiscalPeriod
	highestNumber := 0
	for _, period := range periods {
		if period == nil {
			continue
		}
		if period.PeriodNumber > highestNumber {
			highestNumber = period.PeriodNumber
		}
		if period.IsAdjusting || period.PeriodType == fiscalperiod.PeriodTypeAdjusting {
			if adjusting == nil || period.PeriodNumber > adjusting.PeriodNumber {
				adjusting = period
			}
			continue
		}
		if lastOperating == nil || period.PeriodNumber > lastOperating.PeriodNumber {
			lastOperating = period
		}
	}

	if adjusting != nil {
		if adjusting.Status == fiscalperiod.StatusPermanentlyClosed {
			return periodTarget{}, accountingBlocker(
				"periods",
				fmt.Sprintf(
					"%s is permanently closed and cannot take the year-end closing entry.",
					adjusting.Name,
				),
			), nil
		}

		return periodTarget{
			ID:             adjusting.ID,
			Name:           adjusting.Name,
			Status:         adjusting.Status,
			PeriodNumber:   adjusting.PeriodNumber,
			FiscalYearID:   fy.ID,
			AccountingDate: fy.EndDate,
			StartDate:      adjusting.StartDate,
			EndDate:        adjusting.EndDate,
		}, nil, nil
	}

	if lastOperating == nil {
		return periodTarget{}, accountingBlocker(
			"periods",
			"This fiscal year has only adjusting periods, so the closing entry has no date range to use.",
		), nil
	}

	nextNumber := highestNumber + 1
	if nextNumber > fiscalperiod.MaxAdjustingPeriodNumber {
		return periodTarget{}, accountingBlocker(
			"periods",
			fmt.Sprintf(
				"This fiscal year already uses period %d; there is no ordinal left for a year-end adjusting period.",
				highestNumber,
			),
		), nil
	}

	return periodTarget{
		Name:           fmt.Sprintf("Adjusting Period - %s", fy.Name),
		PeriodNumber:   nextNumber,
		FiscalYearID:   fy.ID,
		AccountingDate: fy.EndDate,
		MustCreate:     true,
		StartDate:      lastOperating.StartDate,
		EndDate:        lastOperating.EndDate,
	}, nil, nil
}

// resolveCarryforwardTarget finds the first period of the following fiscal year.
// The next year has to exist and still be open: without it there is nowhere for
// the closing balances to land, and a year closed without carryforward starts
// the new year from zero assets.
func (s *Service) resolveCarryforwardTarget(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
) (*fiscalyear.FiscalYear, periodTarget, []*fiscalclose.Blocker, error) {
	blockers := make([]*fiscalclose.Blocker, 0, 1)

	nextYear, err := s.fiscalYearRepo.GetNextFiscalYear(
		ctx,
		repositories.GetNextFiscalYearRequest{
			TenantInfo: tenantOf(fy),
			AfterDate:  fy.EndDate,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, periodTarget{}, appendBlocker(blockers, accountingBlocker(
				"nextFiscalYear",
				fmt.Sprintf(
					"Create the fiscal year that follows %s before closing it, so closing balances have somewhere to carry forward to.",
					fy.Name,
				),
			)), nil
		}
		return nil, periodTarget{}, nil, err
	}

	if nextYear.Status == fiscalyear.StatusClosed ||
		nextYear.Status == fiscalyear.StatusPermanentlyClosed {
		return nextYear, periodTarget{}, appendBlocker(blockers, accountingBlocker(
			"nextFiscalYear",
			fmt.Sprintf(
				"%s is already %s, so opening balances cannot be carried into it.",
				nextYear.Name,
				nextYear.Status,
			),
		)), nil
	}

	periods, err := s.fiscalPeriodRepo.ListByFiscalYearID(
		ctx,
		repositories.ListByFiscalYearIDRequest{
			FiscalYearID: nextYear.ID,
			OrgID:        fy.OrganizationID,
			BuID:         fy.BusinessUnitID,
		},
	)
	if err != nil {
		return nil, periodTarget{}, nil, err
	}

	var first *fiscalperiod.FiscalPeriod
	for _, period := range periods {
		if period == nil || period.IsAdjusting ||
			period.PeriodType == fiscalperiod.PeriodTypeAdjusting {
			continue
		}
		if first == nil || period.PeriodNumber < first.PeriodNumber {
			first = period
		}
	}

	if first == nil {
		return nextYear, periodTarget{}, appendBlocker(blockers, accountingBlocker(
			"nextFiscalYear",
			fmt.Sprintf("%s has no operating periods to carry opening balances into.", nextYear.Name),
		)), nil
	}

	if first.Status == fiscalperiod.StatusPermanentlyClosed {
		return nextYear, periodTarget{}, appendBlocker(blockers, accountingBlocker(
			"nextFiscalYear",
			fmt.Sprintf("%s is permanently closed and cannot take opening balances.", first.Name),
		)), nil
	}

	return nextYear, periodTarget{
		ID:             first.ID,
		Name:           first.Name,
		PeriodNumber:   first.PeriodNumber,
		FiscalYearID:   nextYear.ID,
		AccountingDate: nextYear.StartDate,
		StartDate:      first.StartDate,
		EndDate:        first.EndDate,
	}, blockers, nil
}

// closeRevision counts how many times this year has already been closed. A
// reopened year is closed again with fresh idempotency keys, and the revision is
// what keeps those keys unique.
func (s *Service) closeRevision(ctx context.Context, fy *fiscalyear.FiscalYear) (int, error) {
	sources, err := s.sourceRepo.ListByObject(ctx, repositories.GetJournalSourceByObjectRequest{
		TenantInfo:       tenantOf(fy),
		SourceObjectType: sourceObjectType,
		SourceObjectID:   fy.ID.String(),
	})
	if err != nil {
		return 0, err
	}

	revision := 0
	for _, source := range sources {
		if source != nil && source.SourceEventType == sourceEventClosed {
			revision++
		}
	}

	return revision, nil
}

func tenantOf(fy *fiscalyear.FiscalYear) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: fy.OrganizationID, BuID: fy.BusinessUnitID}
}

func accountingBlocker(field, message string) *fiscalclose.Blocker {
	return &fiscalclose.Blocker{
		Field:    field,
		Code:     errortypes.ErrInvalid,
		Message:  message,
		Category: "accounting",
	}
}

func appendBlocker(
	blockers []*fiscalclose.Blocker,
	blocker *fiscalclose.Blocker,
) []*fiscalclose.Blocker {
	if blocker == nil {
		return blockers
	}

	return append(blockers, blocker)
}

func blockerError(blockers []*fiscalclose.Blocker) *errortypes.MultiError {
	if len(blockers) == 0 {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	for _, blocker := range blockers {
		if blocker == nil {
			continue
		}
		multiErr.Add(blocker.Field, blocker.Code, blocker.Message)
	}

	return multiErr
}
