package journalentry

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/journalentryvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger           *zap.Logger
	Repo             repositories.JournalEntryRepository
	GLAccountRepo    repositories.GLAccountRepository
	AuditService     services.AuditService
	PermissionEngine ports.PermissionEngine
	Validator        *journalentryvalidator.Validator
}

type Service struct {
	l             *zap.Logger
	repo          repositories.JournalEntryRepository
	glAccountRepo repositories.GLAccountRepository
	pe            ports.PermissionEngine
	as            services.AuditService
	v             *journalentryvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:             p.Logger.Named("service.journalentry"),
		repo:          p.Repo,
		glAccountRepo: p.GLAccountRepo,
		pe:            p.PermissionEngine,
		as:            p.AuditService,
		v:             p.Validator,
	}
}

// List retrieves a paginated list of journal entries
func (s *Service) List(
	ctx context.Context,
	req *repositories.ListJournalEntryRequest,
) (*pagination.ListResult[*accounting.JournalEntry], error) {
	return s.repo.List(ctx, req)
}

// Get retrieves a single journal entry by ID
func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetJournalEntryByIDRequest,
) (*accounting.JournalEntry, error) {
	return s.repo.GetByID(ctx, req)
}

// GetByNumber retrieves a journal entry by entry number
func (s *Service) GetByNumber(
	ctx context.Context,
	req *repositories.GetJournalEntryByNumberRequest,
) (*accounting.JournalEntry, error) {
	return s.repo.GetByNumber(ctx, req)
}

// GetByReference retrieves journal entries by reference
func (s *Service) GetByReference(
	ctx context.Context,
	req *repositories.GetJournalEntriesByReferenceRequest,
) ([]*accounting.JournalEntry, error) {
	return s.repo.GetByReference(ctx, req)
}

// GetByPeriod retrieves journal entries for a fiscal period
func (s *Service) GetByPeriod(
	ctx context.Context,
	req *repositories.GetJournalEntriesByPeriodRequest,
) ([]*accounting.JournalEntry, error) {
	return s.repo.GetByPeriod(ctx, req)
}

// Create creates a new journal entry with lines
func (s *Service) Create(
	ctx context.Context,
	entry *accounting.JournalEntry,
	lines []*accounting.JournalEntryLine,
	userID pulid.ID,
) (*accounting.JournalEntry, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("buID", entry.BusinessUnitID.String()),
		zap.String("orgID", entry.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	// Set created by
	entry.CreatedByID = userID

	// Generate entry number if not provided
	if entry.EntryNumber == "" {
		entryNumber, err := s.generateEntryNumber(ctx, entry.OrganizationID, entry.BusinessUnitID)
		if err != nil {
			log.Error("failed to generate entry number", zap.Error(err))
			return nil, err
		}
		entry.EntryNumber = entryNumber
	}

	// Calculate totals from lines
	entry.TotalDebit, entry.TotalCredit = s.calculateTotals(lines)

	// Set lines on entry for validation
	entry.Lines = lines

	// Validate
	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}
	if err := s.v.Validate(ctx, valCtx, entry); err != nil {
		return nil, err
	}

	// Create entry with lines
	createdEntry, err := s.repo.CreateWithLines(ctx, entry, lines)
	if err != nil {
		log.Error("failed to create journal entry", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceJournalEntry,
			ResourceID:     createdEntry.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntry),
			OrganizationID: createdEntry.OrganizationID,
			BusinessUnitID: createdEntry.BusinessUnitID,
		},
		audit.WithComment(fmt.Sprintf("Journal entry %s created", createdEntry.EntryNumber)),
	)
	if err != nil {
		log.Error("failed to log journal entry creation", zap.Error(err))
	}

	return createdEntry, nil
}

// Update updates an existing journal entry with lines
func (s *Service) Update(
	ctx context.Context,
	entry *accounting.JournalEntry,
	lines []*accounting.JournalEntryLine,
	userID pulid.ID,
) (*accounting.JournalEntry, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entry.ID.String()),
		zap.String("userID", userID.String()),
	)

	// Get original for audit
	original, err := s.repo.GetByID(ctx, &repositories.GetJournalEntryByIDRequest{
		JournalEntryID: entry.ID,
		OrgID:          entry.OrganizationID,
		BuID:           entry.BusinessUnitID,
		FilterOptions: repositories.JournalEntryFilterOptions{
			IncludeLines: true,
		},
	})
	if err != nil {
		return nil, err
	}

	// Set updated by
	entry.UpdatedByID = &userID

	// Calculate totals from lines
	entry.TotalDebit, entry.TotalCredit = s.calculateTotals(lines)

	// Set lines on entry for validation
	entry.Lines = lines

	// Validate
	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}
	if err := s.v.Validate(ctx, valCtx, entry); err != nil {
		return nil, err
	}

	// Update entry with lines
	updatedEntry, err := s.repo.UpdateWithLines(ctx, entry, lines)
	if err != nil {
		log.Error("failed to update journal entry", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceJournalEntry,
			ResourceID:     updatedEntry.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntry),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntry.OrganizationID,
			BusinessUnitID: updatedEntry.BusinessUnitID,
		},
		audit.WithComment(fmt.Sprintf("Journal entry %s updated", updatedEntry.EntryNumber)),
		audit.WithDiff(original, updatedEntry),
	)
	if err != nil {
		log.Error("failed to log journal entry update", zap.Error(err))
	}

	return updatedEntry, nil
}

// Delete deletes a journal entry (only if in Draft status)
func (s *Service) Delete(
	ctx context.Context,
	req *repositories.DeleteJournalEntryRequest,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("journalEntryId", req.JournalEntryID.String()),
	)

	// Get existing entry
	existing, err := s.repo.GetByID(ctx, &repositories.GetJournalEntryByIDRequest{
		JournalEntryID: req.JournalEntryID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
	})
	if err != nil {
		return err
	}

	// Only allow deletion of draft entries
	if existing.Status != accounting.JournalEntryStatusDraft {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only draft journal entries can be deleted",
		)
	}

	// Delete
	if err := s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete journal entry", zap.Error(err))
		return err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceJournalEntry,
			ResourceID:     existing.GetID(),
			Operation:      permission.OpDelete,
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			UserID:         req.UserID,
		},
		audit.WithComment(fmt.Sprintf("Journal entry %s deleted", existing.EntryNumber)),
		audit.WithDiff(existing, nil),
	)
	if err != nil {
		log.Error("failed to log journal entry deletion", zap.Error(err))
	}

	return nil
}

// Post posts a journal entry to the general ledger
func (s *Service) Post(
	ctx context.Context,
	req *repositories.PostJournalEntryRequest,
) (*accounting.JournalEntry, error) {
	log := s.l.With(
		zap.String("operation", "Post"),
		zap.String("journalEntryId", req.JournalEntryID.String()),
	)

	// Get entry with lines
	entry, err := s.repo.GetByID(ctx, &repositories.GetJournalEntryByIDRequest{
		JournalEntryID: req.JournalEntryID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
		FilterOptions: repositories.JournalEntryFilterOptions{
			IncludeLines: true,
		},
	})
	if err != nil {
		return nil, err
	}

	// Validate can be posted
	if !entry.CanBePosted() {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"__all__",
			errortypes.ErrInvalid,
			"Journal entry cannot be posted. Check balance and approval status.",
		)
		return nil, multiErr
	}

	// Set posted timestamp if not provided
	if req.PostedAt == 0 {
		req.PostedAt = utils.NowUnix()
	}

	// Post the entry
	postedEntry, err := s.repo.Post(ctx, req)
	if err != nil {
		log.Error("failed to post journal entry", zap.Error(err))
		return nil, err
	}

	// Update GL account balances
	if err = s.updateGLAccountBalances(ctx, entry); err != nil {
		log.Error("failed to update GL account balances", zap.Error(err))
		// TODO: Consider rollback strategy
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceJournalEntry,
			ResourceID:     postedEntry.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(postedEntry),
			PreviousState:  jsonutils.MustToJSON(entry),
			OrganizationID: postedEntry.OrganizationID,
			BusinessUnitID: postedEntry.BusinessUnitID,
		},
		audit.WithComment(fmt.Sprintf("Journal entry %s posted", postedEntry.EntryNumber)),
		audit.WithDiff(entry, postedEntry),
	)
	if err != nil {
		log.Error("failed to log journal entry posting", zap.Error(err))
	}

	return postedEntry, nil
}

// Approve approves a journal entry
func (s *Service) Approve(
	ctx context.Context,
	req *repositories.ApproveJournalEntryRequest,
) (*accounting.JournalEntry, error) {
	log := s.l.With(
		zap.String("operation", "Approve"),
		zap.String("journalEntryId", req.JournalEntryID.String()),
	)

	// Get existing entry
	existing, err := s.repo.GetByID(ctx, &repositories.GetJournalEntryByIDRequest{
		JournalEntryID: req.JournalEntryID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
	})
	if err != nil {
		return nil, err
	}

	// Set approved timestamp if not provided
	if req.ApprovedAt == 0 {
		req.ApprovedAt = utils.NowUnix()
	}

	// Approve
	approvedEntry, err := s.repo.Approve(ctx, req)
	if err != nil {
		log.Error("failed to approve journal entry", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceJournalEntry,
			ResourceID:     approvedEntry.GetID(),
			Operation:      permission.OpApprove,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(approvedEntry),
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: approvedEntry.OrganizationID,
			BusinessUnitID: approvedEntry.BusinessUnitID,
		},
		audit.WithComment(fmt.Sprintf("Journal entry %s approved", approvedEntry.EntryNumber)),
		audit.WithDiff(existing, approvedEntry),
	)
	if err != nil {
		log.Error("failed to log journal entry approval", zap.Error(err))
	}

	return approvedEntry, nil
}

// Reject rejects a journal entry
func (s *Service) Reject(
	ctx context.Context,
	req *repositories.RejectJournalEntryRequest,
) (*accounting.JournalEntry, error) {
	log := s.l.With(
		zap.String("operation", "Reject"),
		zap.String("journalEntryId", req.JournalEntryID.String()),
	)

	// Get existing entry
	existing, err := s.repo.GetByID(ctx, &repositories.GetJournalEntryByIDRequest{
		JournalEntryID: req.JournalEntryID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
	})
	if err != nil {
		return nil, err
	}

	// Reject
	rejectedEntry, err := s.repo.Reject(ctx, req)
	if err != nil {
		log.Error("failed to reject journal entry", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceJournalEntry,
			ResourceID:     rejectedEntry.GetID(),
			Operation:      permission.OpReject,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(rejectedEntry),
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: rejectedEntry.OrganizationID,
			BusinessUnitID: rejectedEntry.BusinessUnitID,
		},
		audit.WithComment(
			fmt.Sprintf(
				"Journal entry %s rejected: %s",
				rejectedEntry.EntryNumber,
				req.RejectionNotes,
			),
		),
		audit.WithDiff(existing, rejectedEntry),
	)
	if err != nil {
		log.Error("failed to log journal entry rejection", zap.Error(err))
	}

	return rejectedEntry, nil
}

// Reverse reverses a posted journal entry
func (s *Service) Reverse(
	ctx context.Context,
	req *repositories.ReverseJournalEntryRequest,
) (*accounting.JournalEntry, error) {
	log := s.l.With(
		zap.String("operation", "Reverse"),
		zap.String("journalEntryId", req.JournalEntryID.String()),
	)

	// Get existing entry
	existing, err := s.repo.GetByID(ctx, &repositories.GetJournalEntryByIDRequest{
		JournalEntryID: req.JournalEntryID,
		OrgID:          req.OrgID,
		BuID:           req.BuID,
	})
	if err != nil {
		return nil, err
	}

	// Validate can be reversed
	if !existing.CanBeReversed() {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"__all__",
			errortypes.ErrInvalid,
			"Journal entry cannot be reversed. It must be posted and not already reversed.",
		)
		return nil, multiErr
	}

	// Set reversal date if not provided
	if req.ReversalDate == 0 {
		req.ReversalDate = utils.NowUnix()
	}

	// Reverse (this creates a new reversal entry)
	reversalEntry, err := s.repo.Reverse(ctx, req)
	if err != nil {
		log.Error("failed to reverse journal entry", zap.Error(err))
		return nil, err
	}

	// Audit log for original entry
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceJournalEntry,
			ResourceID:     existing.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: existing.OrganizationID,
			BusinessUnitID: existing.BusinessUnitID,
		},
		audit.WithComment(
			fmt.Sprintf("Journal entry %s reversed: %s", existing.EntryNumber, req.ReversalReason),
		),
	)
	if err != nil {
		log.Error("failed to log journal entry reversal", zap.Error(err))
	}

	// Audit log for reversal entry
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceJournalEntry,
			ResourceID:     reversalEntry.GetID(),
			Operation:      permission.OpCreate,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(reversalEntry),
			OrganizationID: reversalEntry.OrganizationID,
			BusinessUnitID: reversalEntry.BusinessUnitID,
		},
		audit.WithComment(
			fmt.Sprintf(
				"Reversal entry %s created for %s",
				reversalEntry.EntryNumber,
				existing.EntryNumber,
			),
		),
	)
	if err != nil {
		log.Error("failed to log reversal entry creation", zap.Error(err))
	}

	return reversalEntry, nil
}

// calculateTotals calculates total debits and credits from lines
func (s *Service) calculateTotals(
	lines []*accounting.JournalEntryLine,
) (totalDebit, totalCredit int64) {
	for _, line := range lines {
		totalDebit += line.DebitAmount
		totalCredit += line.CreditAmount
	}
	return totalDebit, totalCredit
}

// generateEntryNumber generates a unique entry number
func (s *Service) generateEntryNumber(ctx context.Context, orgID, buID pulid.ID) (string, error) {
	// TODO: Implement proper entry number generation with sequence
	// For now, use a simple format: JE-YYYY-NNNNNN
	// In production, this should use a database sequence or counter
	return fmt.Sprintf("JE-%d-%06d", utils.NowUnix()/31536000+1970, utils.NowUnix()%1000000), nil
}

// updateGLAccountBalances updates GL account balances after posting
func (s *Service) updateGLAccountBalances(
	ctx context.Context,
	entry *accounting.JournalEntry,
) error {
	log := s.l.With(
		zap.String("operation", "updateGLAccountBalances"),
		zap.String("journalEntryId", entry.ID.String()),
	)

	// Group lines by GL account
	accountUpdates := make(map[string]struct {
		debit  int64
		credit int64
	})

	for _, line := range entry.Lines {
		accountID := line.GLAccountID.String()
		update := accountUpdates[accountID]
		update.debit += line.DebitAmount
		update.credit += line.CreditAmount
		accountUpdates[accountID] = update
	}

	// Update each account
	for accountIDStr, update := range accountUpdates {
		accountID, _ := pulid.Parse(accountIDStr)

		// Get current account
		account, err := s.glAccountRepo.GetByID(ctx, &repositories.GetGLAccountByIDRequest{
			GLAccountID: accountID,
			OrgID:       entry.OrganizationID,
			BuID:        entry.BusinessUnitID,
		})
		if err != nil {
			log.Error(
				"failed to get GL account",
				zap.Error(err),
				zap.String("accountId", accountIDStr),
			)
			return err
		}

		// Calculate new balance
		newBalance := account.CurrentBalance + update.debit - update.credit

		// Update balance
		_, err = s.glAccountRepo.UpdateBalance(ctx, &repositories.UpdateGLAccountBalanceRequest{
			GLAccountID:    accountID,
			OrgID:          entry.OrganizationID,
			BuID:           entry.BusinessUnitID,
			DebitAmount:    update.debit,
			CreditAmount:   update.credit,
			CurrentBalance: newBalance,
		})
		if err != nil {
			log.Error(
				"failed to update GL account balance",
				zap.Error(err),
				zap.String("accountId", accountIDStr),
			)
			return err
		}
	}

	return nil
}
