package journalreviewservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/journalposting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	DB               ports.DBConnection
	Repo             repositories.JournalReviewRepository
	AccountingRepo   repositories.AccountingControlRepository
	FiscalPeriodRepo repositories.FiscalPeriodRepository
	AuditService     serviceports.AuditService
	Realtime         serviceports.RealtimeService `optional:"true"`
}

type Service struct {
	l        *zap.Logger
	db       ports.DBConnection
	repo     repositories.JournalReviewRepository
	controls repositories.AccountingControlRepository
	periods  repositories.FiscalPeriodRepository
	audit    serviceports.AuditService
	realtime serviceports.RealtimeService
	now      func() int64
}

var _ serviceports.JournalReviewService = (*Service)(nil)

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:        p.Logger.Named("service.journal-review"),
		db:       p.DB,
		repo:     p.Repo,
		controls: p.AccountingRepo,
		periods:  p.FiscalPeriodRepo,
		audit:    p.AuditService,
		realtime: p.Realtime,
		now:      timeutils.NowUnix,
	}
}

type reviewStep func(
	ctx context.Context,
	entry *journalentry.JournalEntry,
	req *serviceports.JournalReviewRequest,
	actorID pulid.ID,
	now int64,
) error

func (s *Service) Approve(
	ctx context.Context,
	req *serviceports.JournalReviewRequest,
	actor *serviceports.RequestActor,
) (*serviceports.JournalReviewResult, error) {
	return s.review(ctx, req, actor, "Approved journal entry", s.approveEntry)
}

func (s *Service) Post(
	ctx context.Context,
	req *serviceports.JournalReviewRequest,
	actor *serviceports.RequestActor,
) (*serviceports.JournalReviewResult, error) {
	return s.review(ctx, req, actor, "Posted journal entry to the general ledger", s.postEntry)
}

func (s *Service) review(
	ctx context.Context,
	req *serviceports.JournalReviewRequest,
	actor *serviceports.RequestActor,
	comment string,
	step reviewStep,
) (*serviceports.JournalReviewResult, error) {
	entryIDs, err := validateRequest(req, actor)
	if err != nil {
		return nil, err
	}

	result := &serviceports.JournalReviewResult{
		Outcomes: make([]*serviceports.JournalReviewOutcome, 0, len(entryIDs)),
	}
	for _, entryID := range entryIDs {
		outcome, previous, current, stepErr := s.reviewOne(ctx, req, entryID, actor.UserID, step)
		if stepErr != nil {
			return nil, stepErr
		}
		result.Outcomes = append(result.Outcomes, outcome)
		if outcome.Error != "" {
			result.Failed++
			continue
		}
		result.Changed++
		s.record(ctx, previous, current, actor.UserID, comment)
	}
	return result, nil
}

func (s *Service) reviewOne(
	ctx context.Context,
	req *serviceports.JournalReviewRequest,
	entryID pulid.ID,
	actorID pulid.ID,
	step reviewStep,
) (outcome *serviceports.JournalReviewOutcome, previous, current *journalentry.JournalEntry, err error) {
	outcome = &serviceports.JournalReviewOutcome{EntryID: entryID}

	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entry, lockErr := s.repo.LockEntry(txCtx, repositories.LockJournalEntryRequest{
			TenantInfo: req.TenantInfo,
			EntryID:    entryID,
		})
		if lockErr != nil {
			return lockErr
		}
		outcome.EntryNumber = entry.EntryNumber
		outcome.Status = string(entry.Status)
		snapshot := *entry
		previous = &snapshot

		if stepErr := step(txCtx, entry, req, actorID, s.now()); stepErr != nil {
			return stepErr
		}
		outcome.Status = string(entry.Status)
		outcome.Changed = true
		current = entry
		return nil
	})
	if err == nil {
		return outcome, previous, current, nil
	}
	if isRefusal(err) {
		outcome.Changed = false
		outcome.Error = refusalMessage(ctx, err)
		return outcome, nil, nil, nil
	}
	return nil, nil, nil, fmt.Errorf("review journal entry %s: %w", entryID, err)
}

func (s *Service) approveEntry(
	ctx context.Context,
	entry *journalentry.JournalEntry,
	req *serviceports.JournalReviewRequest,
	actorID pulid.ID,
	now int64,
) error {
	if err := entry.CanApprove(); err != nil {
		return err
	}
	if err := s.repo.ApproveEntry(ctx, &repositories.ApproveJournalEntryParams{
		TenantInfo:   req.TenantInfo,
		EntryID:      entry.ID,
		BatchID:      entry.BatchID,
		Version:      entry.Version,
		ApprovedByID: actorID,
		ApprovedAt:   now,
	}); err != nil {
		return err
	}

	entry.Status = journalentry.StatusApproved
	entry.IsApproved = true
	entry.ApprovedByID = actorID
	entry.ApprovedAt = &now
	entry.Version++
	return nil
}

func (s *Service) postEntry(
	ctx context.Context,
	entry *journalentry.JournalEntry,
	req *serviceports.JournalReviewRequest,
	actorID pulid.ID,
	now int64,
) error {
	if err := entry.CanPost(); err != nil {
		return err
	}

	control, err := s.controls.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return err
	}
	resolved, err := journalposting.ResolvePeriod(
		ctx,
		s.periods,
		&journalposting.ResolvePeriodRequest{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			Date:           entry.AccountingDate,
			Policy:         control.ClosedPeriodPostingPolicy,
			Subject:        "journal entry",
		},
	)
	if err != nil {
		return err
	}

	if err = s.repo.PostEntry(ctx, &repositories.PostJournalEntryParams{
		TenantInfo:     req.TenantInfo,
		EntryID:        entry.ID,
		BatchID:        entry.BatchID,
		Version:        entry.Version,
		FiscalYearID:   resolved.Period.FiscalYearID,
		FiscalPeriodID: resolved.Period.ID,
		AccountingDate: resolved.AccountingDate,
		PostedByID:     actorID,
		PostedAt:       now,
		Lines:          postingLines(entry.Lines),
	}); err != nil {
		return err
	}

	entry.Status = journalentry.StatusPosted
	entry.IsPosted = true
	entry.PostedByID = actorID
	entry.PostedAt = &now
	entry.FiscalYearID = resolved.Period.FiscalYearID
	entry.FiscalPeriodID = resolved.Period.ID
	entry.AccountingDate = resolved.AccountingDate
	entry.Version++
	return nil
}

func postingLines(lines []*journalentry.JournalEntryLine) []repositories.JournalPostingLine {
	out := make([]repositories.JournalPostingLine, 0, len(lines))
	for _, line := range lines {
		if line == nil {
			continue
		}
		out = append(out, repositories.JournalPostingLine{
			ID:           line.ID,
			GLAccountID:  line.GLAccountID,
			LineNumber:   line.LineNumber,
			Description:  line.Description,
			DebitAmount:  line.DebitAmount,
			CreditAmount: line.CreditAmount,
			NetAmount:    line.NetAmount,
			CustomerID:   line.CustomerID,
			LocationID:   line.LocationID,
		})
	}
	return out
}

func validateRequest(
	req *serviceports.JournalReviewRequest,
	actor *serviceports.RequestActor,
) ([]pulid.ID, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Approving and posting journal entries needs a signed-in person",
		)
	}
	if req == nil || req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return nil, errortypes.NewValidationError(
			"tenantInfo",
			errortypes.ErrRequired,
			"The organization and business unit are required",
		)
	}

	entryIDs := make([]pulid.ID, 0, len(req.EntryIDs))
	for _, id := range req.EntryIDs {
		if id.IsNil() || slices.Contains(entryIDs, id) {
			continue
		}
		entryIDs = append(entryIDs, id)
	}
	switch {
	case len(entryIDs) == 0:
		return nil, errortypes.NewValidationError(
			"entryIds",
			errortypes.ErrRequired,
			"Choose at least one journal entry",
		)
	case len(entryIDs) > serviceports.MaxJournalReviewEntries:
		return nil, errortypes.NewValidationError(
			"entryIds",
			errortypes.ErrInvalid,
			"Choose at most {0} journal entries at a time",
			serviceports.MaxJournalReviewEntries,
		)
	}
	return entryIDs, nil
}

func isRefusal(err error) bool {
	return errortypes.IsBusinessError(err) ||
		errortypes.IsConflictError(err) ||
		errortypes.IsNotFoundError(err) ||
		errortypes.IsError(err)
}

type localizedError interface {
	LocalizedMessage() (message string, args []any)
}

func refusalMessage(ctx context.Context, err error) string {
	if errortypes.IsNotFoundError(err) {
		return i18n.T(ctx, "Journal entry not found")
	}
	var localized localizedError
	if errors.As(err, &localized) {
		message, args := localized.LocalizedMessage()
		return i18n.T(ctx, message, args...)
	}
	return err.Error()
}

func (s *Service) record(
	ctx context.Context,
	previous, current *journalentry.JournalEntry,
	actorID pulid.ID,
	comment string,
) {
	if current == nil {
		return
	}
	if s.realtime != nil {
		if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
			OrganizationID: current.OrganizationID,
			BusinessUnitID: current.BusinessUnitID,
			ActorUserID:    actorID,
			Resource:       permission.ResourceJournalEntry.String(),
			Action:         string(permission.OpApprove),
			RecordID:       current.ID,
		}); err != nil {
			s.l.Warn("failed to publish journal entry invalidation", zap.Error(err))
		}
	}
	if s.audit == nil {
		return
	}

	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceJournalEntry,
		ResourceID:     current.ID.String(),
		Operation:      permission.OpApprove,
		UserID:         actorID,
		CurrentState:   auditState(current),
		PreviousState:  auditState(previous),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
		Critical:       current.Status == journalentry.StatusPosted,
	}
	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log journal review audit action", zap.Error(err))
	}
}

func auditState(entry *journalentry.JournalEntry) map[string]any {
	if entry == nil {
		return nil
	}
	return map[string]any{
		"entryNumber":    entry.EntryNumber,
		"status":         string(entry.Status),
		"isApproved":     entry.IsApproved,
		"approvedById":   entry.ApprovedByID.String(),
		"isPosted":       entry.IsPosted,
		"postedById":     entry.PostedByID.String(),
		"fiscalPeriodId": entry.FiscalPeriodID.String(),
		"accountingDate": entry.AccountingDate,
		"totalDebit":     entry.TotalDebit,
		"totalCredit":    entry.TotalCredit,
		"version":        entry.Version,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *serviceports.ListJournalReviewRequest,
) (*pagination.CursorListResult[*journalentry.JournalEntry], error) {
	if req.Filter == nil {
		req.Filter = &pagination.QueryOptions{}
	}
	req.Filter.TenantInfo = req.TenantInfo
	return s.repo.ListConnection(ctx, &repositories.ListJournalReviewRequest{
		Filter:   req.Filter,
		Cursor:   req.Cursor,
		Statuses: req.Statuses,
	})
}

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*serviceports.JournalReviewSummary, error) {
	counts, err := s.repo.Summarize(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	summary := &serviceports.JournalReviewSummary{
		AwaitingApproval:     counts.AwaitingApproval,
		ReadyToPost:          counts.ReadyToPost,
		OldestAccountingDate: counts.OldestAccountingDate,
		PostingMode:          tenant.JournalPostingModeAutomatic,
	}

	control, err := s.controls.GetByOrgID(ctx, tenantInfo.OrgID)
	switch {
	case errortypes.IsNotFoundError(err):
		return summary, nil
	case err != nil:
		return nil, err
	}
	summary.PostingMode = control.JournalPostingMode
	summary.RequiresApproval = control.RequireManualJEApproval
	return summary, nil
}
