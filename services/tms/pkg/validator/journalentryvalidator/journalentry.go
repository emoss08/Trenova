package journalentryvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	factory *framework.TenantedValidatorFactory[*accounting.JournalEntry]
	getDB   func(context.Context) (*bun.DB, error)
}

func NewValidator(p Params) *Validator {
	getDB := func(ctx context.Context) (*bun.DB, error) {
		return p.DB.DB(ctx)
	}

	factory := framework.NewTenantedValidatorFactory[*accounting.JournalEntry](
		getDB,
	).
		WithModelName("JournalEntry").
		WithUniqueFields(func(je *accounting.JournalEntry) []framework.UniqueField {
			return []framework.UniqueField{
				{
					Name:     "entry_number",
					GetValue: func() string { return je.EntryNumber },
					Message:  "Entry number ':value' already exists in the organization.",
				},
			}
		}).
		WithCustomRules(
			func(entity *accounting.JournalEntry, vc *validator.ValidationContext) []framework.ValidationRule {
				var rules []framework.ValidationRule

				if vc.IsCreate {
					rules = append(rules, framework.NewBusinessRule("id_validation").
						WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
							if entity.ID.IsNotNil() {
								multiErr.Add(
									"id",
									errortypes.ErrInvalid,
									"ID cannot be set on create",
								)
							}
							return nil
						}),
					)
				}

				rules = append(rules,
					// Line validation
					framework.NewBusinessRule("journal_entry_lines_validation").
						WithStage(framework.ValidationStageCompliance).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
							validateLines(entity, me)
							return nil
						}),

					// Balance validation
					framework.NewBusinessRule("journal_entry_balance").
						WithStage(framework.ValidationStageCompliance).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
							validateBalance(entity, me)
							return nil
						}),

					// Fiscal period validation
					framework.NewBusinessRule("fiscal_period_validation").
						WithStage(framework.ValidationStageDataIntegrity).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
							validateFiscalPeriod(ctx, entity, me, getDB)
							return nil
						}),

					// Status transition validation
					framework.NewBusinessRule("status_transition_validation").
						WithStage(framework.ValidationStageCompliance).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
							if vc.IsUpdate {
								validateStatusTransition(ctx, entity, me, getDB)
							}
							return nil
						}),

					// Posting validation
					framework.NewBusinessRule("posting_validation").
						WithStage(framework.ValidationStageCompliance).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
							validatePostingRules(entity, me)
							return nil
						}),

					// Approval validation
					framework.NewBusinessRule("approval_validation").
						WithStage(framework.ValidationStageCompliance).
						WithPriority(framework.ValidationPriorityMedium).
						WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
							validateApprovalRules(entity, me)
							return nil
						}),

					// Reversal validation
					framework.NewBusinessRule("reversal_validation").
						WithStage(framework.ValidationStageDataIntegrity).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
							if entity.IsReversal {
								validateReversalRules(ctx, entity, me, getDB)
							}
							return nil
						}),

					// GL Account validation
					framework.NewBusinessRule("gl_account_validation").
						WithStage(framework.ValidationStageDataIntegrity).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
							validateGLAccounts(ctx, entity, me, getDB)
							return nil
						}),

					// Reference validation
					framework.NewBusinessRule("reference_validation").
						WithStage(framework.ValidationStageCompliance).
						WithPriority(framework.ValidationPriorityLow).
						WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
							validateReferenceConsistency(entity, me)
							return nil
						}),
				)

				return rules
			},
		)

	return &Validator{
		factory: factory,
		getDB:   getDB,
	}
}

// validateLines ensures journal entry has at least 2 lines and line numbers are sequential
func validateLines(entity *accounting.JournalEntry, me *errortypes.MultiError) {
	if len(entity.Lines) < 2 {
		me.Add("lines", errortypes.ErrInvalid, "Journal entry must have at least 2 lines")
		return
	}

	// Validate line numbers are sequential and unique
	lineNumbers := make(map[int32]bool)
	for i, line := range entity.Lines {
		if line.LineNumber <= 0 {
			me.Add(
				fmt.Sprintf("lines[%d].lineNumber", i),
				errortypes.ErrInvalid,
				"Line number must be positive",
			)
		}

		if lineNumbers[line.LineNumber] {
			me.Add(
				fmt.Sprintf("lines[%d].lineNumber", i),
				errortypes.ErrInvalid,
				fmt.Sprintf("Duplicate line number: %d", line.LineNumber),
			)
		}
		lineNumbers[line.LineNumber] = true

		// Validate each line has either debit or credit (not both, not neither)
		if line.DebitAmount > 0 && line.CreditAmount > 0 {
			me.Add(
				fmt.Sprintf("lines[%d]", i),
				errortypes.ErrInvalid,
				"Line cannot have both debit and credit amounts",
			)
		}
		if line.DebitAmount == 0 && line.CreditAmount == 0 {
			me.Add(
				fmt.Sprintf("lines[%d]", i),
				errortypes.ErrInvalid,
				"Line must have either debit or credit amount",
			)
		}
		if line.DebitAmount < 0 || line.CreditAmount < 0 {
			me.Add(
				fmt.Sprintf("lines[%d]", i),
				errortypes.ErrInvalid,
				"Line amounts cannot be negative",
			)
		}

		// Validate GL account is set
		if line.GLAccountID.IsNil() {
			me.Add(
				fmt.Sprintf("lines[%d].glAccountId", i),
				errortypes.ErrInvalid,
				"GL account is required",
			)
		}

		// Validate description is not empty
		if line.Description == "" {
			me.Add(
				fmt.Sprintf("lines[%d].description", i),
				errortypes.ErrInvalid,
				"Line description is required",
			)
		}
	}
}

// validateBalance ensures total debits equal total credits
func validateBalance(entity *accounting.JournalEntry, me *errortypes.MultiError) {
	// Skip balance check for draft entries
	if entity.Status == accounting.JournalEntryStatusDraft {
		return
	}

	var totalDebit, totalCredit int64
	for _, line := range entity.Lines {
		totalDebit += line.DebitAmount
		totalCredit += line.CreditAmount
	}

	if totalDebit != totalCredit {
		me.Add(
			"__all__",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Journal entry is not balanced. Debits: $%.2f, Credits: $%.2f",
				float64(totalDebit)/100.0,
				float64(totalCredit)/100.0,
			),
		)
	}

	// Validate denormalized totals match
	if entity.TotalDebit != totalDebit {
		me.Add(
			"totalDebit",
			errortypes.ErrInvalid,
			"Total debit does not match sum of line debits",
		)
	}
	if entity.TotalCredit != totalCredit {
		me.Add(
			"totalCredit",
			errortypes.ErrInvalid,
			"Total credit does not match sum of line credits",
		)
	}
}

// validateFiscalPeriod ensures the fiscal period exists, is open, and entry date is within period
func validateFiscalPeriod(
	ctx context.Context,
	entity *accounting.JournalEntry,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	// Check fiscal period exists and is open
	var period accounting.FiscalPeriod
	err = db.NewSelect().
		Model(&period).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fp.id = ?", entity.FiscalPeriodID).
				Where("fp.organization_id = ?", entity.OrganizationID).
				Where("fp.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		me.Add("fiscalPeriodId", errortypes.ErrInvalid, "Fiscal period not found")
		return
	}

	// Check if period is open (unless we're reversing a posted entry)
	if period.Status != accounting.PeriodStatusOpen && !entity.IsReversal {
		me.Add(
			"fiscalPeriodId",
			errortypes.ErrInvalid,
			fmt.Sprintf("Cannot post to %s fiscal period", period.Status),
		)
	}

	// Validate entry date is within period
	if entity.EntryDate < period.StartDate || entity.EntryDate > period.EndDate {
		me.Add(
			"entryDate",
			errortypes.ErrInvalid,
			"Entry date must be within the fiscal period",
		)
	}

	// Validate fiscal year matches
	if entity.FiscalYearID != period.FiscalYearID {
		me.Add(
			"fiscalYearId",
			errortypes.ErrInvalid,
			"Fiscal year does not match fiscal period",
		)
	}
}

// validateStatusTransition ensures valid status transitions
func validateStatusTransition(
	ctx context.Context,
	entity *accounting.JournalEntry,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	// Get current status from database
	var current accounting.JournalEntry
	err = db.NewSelect().
		Model(&current).
		Column("status", "is_posted", "is_approved").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("je.id = ?", entity.ID).
				Where("je.organization_id = ?", entity.OrganizationID).
				Where("je.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		return // New entry, no transition to validate
	}

	// Define valid transitions
	validTransitions := map[accounting.JournalEntryStatus][]accounting.JournalEntryStatus{
		accounting.JournalEntryStatusDraft: {
			accounting.JournalEntryStatusPending,
			accounting.JournalEntryStatusPosted, // Direct post without approval
		},
		accounting.JournalEntryStatusPending: {
			accounting.JournalEntryStatusApproved,
			accounting.JournalEntryStatusRejected,
			accounting.JournalEntryStatusDraft, // Return to draft
		},
		accounting.JournalEntryStatusApproved: {
			accounting.JournalEntryStatusPosted,
			accounting.JournalEntryStatusDraft, // Return to draft
		},
		accounting.JournalEntryStatusRejected: {
			accounting.JournalEntryStatusDraft, // Fix and resubmit
		},
		accounting.JournalEntryStatusPosted: {
			accounting.JournalEntryStatusReversed, // Can only be reversed
		},
		accounting.JournalEntryStatusReversed: {}, // Terminal state
	}

	// Check if transition is valid
	if current.Status != entity.Status {
		allowed := validTransitions[current.Status]
		isValid := false
		for _, allowedStatus := range allowed {
			if entity.Status == allowedStatus {
				isValid = true
				break
			}
		}

		if !isValid {
			me.Add(
				"status",
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"Invalid status transition from %s to %s",
					current.Status,
					entity.Status,
				),
			)
		}
	}

	// Cannot modify posted entries (except to reverse)
	if current.IsPosted && entity.Status != accounting.JournalEntryStatusReversed {
		me.Add("__all__", errortypes.ErrInvalid, "Cannot modify posted journal entries")
	}
}

// validatePostingRules ensures posting requirements are met
func validatePostingRules(entity *accounting.JournalEntry, me *errortypes.MultiError) {
	if entity.IsPosted {
		// Must have posted_at and posted_by_id
		if entity.PostedAt == nil || *entity.PostedAt == 0 {
			me.Add("postedAt", errortypes.ErrInvalid, "Posted date is required for posted entries")
		}
		if entity.PostedByID == nil || entity.PostedByID.IsNil() {
			me.Add(
				"postedById",
				errortypes.ErrInvalid,
				"Posted by user is required for posted entries",
			)
		}

		// Status must be Posted
		if entity.Status != accounting.JournalEntryStatusPosted {
			me.Add("status", errortypes.ErrInvalid, "Status must be Posted for posted entries")
		}

		// Must be balanced
		if entity.TotalDebit != entity.TotalCredit {
			me.Add("__all__", errortypes.ErrInvalid, "Posted entries must be balanced")
		}

		// If requires approval, must be approved
		if entity.RequiresApproval && !entity.IsApproved {
			me.Add("__all__", errortypes.ErrInvalid, "Entry must be approved before posting")
		}
	}
}

// validateApprovalRules ensures approval requirements are met
func validateApprovalRules(entity *accounting.JournalEntry, me *errortypes.MultiError) {
	if entity.IsApproved {
		// Must have approved_at and approved_by_id
		if entity.ApprovedAt == nil || *entity.ApprovedAt == 0 {
			me.Add(
				"approvedAt",
				errortypes.ErrInvalid,
				"Approval date is required for approved entries",
			)
		}
		if entity.ApprovedByID == nil || entity.ApprovedByID.IsNil() {
			me.Add(
				"approvedById",
				errortypes.ErrInvalid,
				"Approver is required for approved entries",
			)
		}

		// Status must be Approved or Posted
		if entity.Status != accounting.JournalEntryStatusApproved &&
			entity.Status != accounting.JournalEntryStatusPosted {
			me.Add(
				"status",
				errortypes.ErrInvalid,
				"Status must be Approved or Posted for approved entries",
			)
		}
	}
}

// validateReversalRules ensures reversal entries are valid
func validateReversalRules(
	ctx context.Context,
	entity *accounting.JournalEntry,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	if entity.ReversalOfID == nil || entity.ReversalOfID.IsNil() {
		me.Add(
			"reversalOfId",
			errortypes.ErrInvalid,
			"Reversal entry must reference original entry",
		)
		return
	}

	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	// Get original entry
	var original accounting.JournalEntry
	err = db.NewSelect().
		Model(&original).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("je.id = ?", entity.ReversalOfID).
				Where("je.organization_id = ?", entity.OrganizationID).
				Where("je.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		me.Add("reversalOfId", errortypes.ErrInvalid, "Original entry not found")
		return
	}

	// Original entry must be posted
	if !original.IsPosted {
		me.Add("reversalOfId", errortypes.ErrInvalid, "Can only reverse posted entries")
	}

	// Original entry must not already be reversed
	if original.Status == accounting.JournalEntryStatusReversed {
		me.Add("reversalOfId", errortypes.ErrInvalid, "Entry has already been reversed")
	}

	// Reversal entry type must be Reversal
	if entity.EntryType != accounting.JournalEntryTypeReversal {
		me.Add(
			"entryType",
			errortypes.ErrInvalid,
			"Entry type must be Reversal for reversal entries",
		)
	}

	// Reversal reason is required
	if entity.ReversalReason == "" {
		me.Add("reversalReason", errortypes.ErrInvalid, "Reversal reason is required")
	}
}

// validateGLAccounts ensures all GL accounts exist, are active, and allow manual entries
func validateGLAccounts(
	ctx context.Context,
	entity *accounting.JournalEntry,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	if len(entity.Lines) == 0 {
		return
	}

	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	// Collect all GL account IDs
	accountIDs := make([]string, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if !line.GLAccountID.IsNil() {
			accountIDs = append(accountIDs, line.GLAccountID.String())
		}
	}

	if len(accountIDs) == 0 {
		return
	}

	// Fetch all accounts in one query
	var accounts []accounting.GLAccount
	err = db.NewSelect().
		Model(&accounts).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.id IN (?)", bun.In(accountIDs)).
				Where("gla.organization_id = ?", entity.OrganizationID).
				Where("gla.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to validate GL accounts")
		return
	}

	// Create map for quick lookup
	accountMap := make(map[string]*accounting.GLAccount)
	for i := range accounts {
		accountMap[accounts[i].ID.String()] = &accounts[i]
	}

	// Validate each line's GL account
	for i, line := range entity.Lines {
		if line.GLAccountID.IsNil() {
			continue
		}

		account, exists := accountMap[line.GLAccountID.String()]
		if !exists {
			me.Add(
				fmt.Sprintf("lines[%d].glAccountId", i),
				errortypes.ErrInvalid,
				"GL account not found",
			)
			continue
		}

		// Check if account is active
		if !account.IsActive {
			me.Add(
				fmt.Sprintf("lines[%d].glAccountId", i),
				errortypes.ErrInvalid,
				fmt.Sprintf("GL account '%s' is inactive", account.AccountCode),
			)
		}

		// Check if manual journal entries are allowed (unless auto-generated)
		if !account.AllowManualJE && !entity.IsAutoGenerated {
			me.Add(
				fmt.Sprintf("lines[%d].glAccountId", i),
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"GL account '%s' does not allow manual journal entries",
					account.AccountCode,
				),
			)
		}

		// Check if project is required
		if account.RequireProject && (line.ProjectID == nil || line.ProjectID.IsNil()) {
			me.Add(
				fmt.Sprintf("lines[%d].projectId", i),
				errortypes.ErrInvalid,
				fmt.Sprintf("GL account '%s' requires a project", account.AccountCode),
			)
		}
	}
}

// validateReferenceConsistency ensures reference fields are consistent
func validateReferenceConsistency(entity *accounting.JournalEntry, me *errortypes.MultiError) {
	// If reference type is set, reference ID should also be set
	if entity.ReferenceType != "" && (entity.ReferenceID == nil || entity.ReferenceID.IsNil()) {
		me.Add(
			"referenceId",
			errortypes.ErrInvalid,
			"Reference ID is required when reference type is set",
		)
	}

	// If reference ID is set, reference type should also be set
	if (entity.ReferenceID != nil && !entity.ReferenceID.IsNil()) && entity.ReferenceType == "" {
		me.Add(
			"referenceType",
			errortypes.ErrInvalid,
			"Reference type is required when reference ID is set",
		)
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *accounting.JournalEntry,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
