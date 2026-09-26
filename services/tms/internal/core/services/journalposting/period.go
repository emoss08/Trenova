package journalposting

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type PeriodReader interface {
	GetPeriodByDate(
		ctx context.Context,
		req repositories.GetPeriodByDateRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	ListByFiscalYearID(
		ctx context.Context,
		req repositories.ListByFiscalYearIDRequest,
	) ([]*fiscalperiod.FiscalPeriod, error)
}

type ResolvePeriodRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Date           int64
	Policy         tenant.ClosedPeriodPostingPolicy
	Subject        string
	Field          string
}

type ResolvedPeriod struct {
	Period         *fiscalperiod.FiscalPeriod
	AccountingDate int64
}

func AcceptsPostings(status fiscalperiod.Status) bool {
	return status == fiscalperiod.StatusOpen || status == fiscalperiod.StatusLocked
}

func ResolvePeriod(
	ctx context.Context,
	periods PeriodReader,
	req *ResolvePeriodRequest,
) (*ResolvedPeriod, error) {
	period, err := periods.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: req.OrganizationID,
		BuID:  req.BusinessUnitID,
		Date:  req.Date,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				req.field(),
				errortypes.ErrInvalid,
				"No fiscal period covers the "+req.Subject+" date",
			)
		}
		return nil, err
	}

	switch {
	case AcceptsPostings(period.Status):
		return &ResolvedPeriod{Period: period, AccountingDate: req.Date}, nil
	case period.Status == fiscalperiod.StatusClosed ||
		period.Status == fiscalperiod.StatusPermanentlyClosed:
		return nextOpenPeriod(ctx, periods, req, period)
	default:
		return nil, errortypes.NewBusinessError(
			"The "+req.Subject+" cannot be posted to an inactive fiscal period",
		).WithParam("fiscalPeriodId", period.ID.String())
	}
}

func nextOpenPeriod(
	ctx context.Context,
	periods PeriodReader,
	req *ResolvePeriodRequest,
	closed *fiscalperiod.FiscalPeriod,
) (*ResolvedPeriod, error) {
	if req.Policy != tenant.ClosedPeriodPostingPolicyPostToNextOpen {
		return nil, errortypes.NewBusinessError(
			"The "+req.Subject+" falls in a closed fiscal period; reopen the period first",
		).WithParam("fiscalPeriodId", closed.ID.String())
	}

	candidates, err := periods.ListByFiscalYearID(ctx, repositories.ListByFiscalYearIDRequest{
		FiscalYearID: closed.FiscalYearID,
		OrgID:        req.OrganizationID,
		BuID:         req.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	var next *fiscalperiod.FiscalPeriod
	for _, candidate := range candidates {
		if candidate == nil || candidate.PeriodNumber <= closed.PeriodNumber ||
			!AcceptsPostings(candidate.Status) {
			continue
		}
		if next == nil || candidate.PeriodNumber < next.PeriodNumber {
			next = candidate
		}
	}
	if next == nil {
		return nil, errortypes.NewBusinessError(
			"No next open fiscal period is available for the "+req.Subject,
		).WithParam("fiscalPeriodId", closed.ID.String())
	}

	return &ResolvedPeriod{Period: next, AccountingDate: next.StartDate}, nil
}

func (req *ResolvePeriodRequest) field() string {
	if req.Field == "" {
		return "accountingDate"
	}
	return req.Field
}
