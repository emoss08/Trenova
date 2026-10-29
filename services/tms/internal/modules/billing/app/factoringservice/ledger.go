package factoringservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalposting"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxFutureMovementSeconds = timeutils.SecondsPerDay

type LedgerParams struct {
	fx.In

	DB          kernelports.Transactor
	Submissions factoringsubmissionports.SubmissionRepository
	Entries     factoringledgerports.ReserveEntryRepository
	Agreements  factoringports.AgreementRepository
	Invoices    invoiceports.InvoiceRepository
	Ledger      customerledgerports.CustomerLedgerProjectionRepository
	Controls    tenantports.AccountingControlRepository
	Periods     fiscalperiodports.FiscalPeriodRepository
	Sequences   seqgen.Generator
	Journals    journalentryports.JournalPostingRepository
	Requester   factoringports.EvaluationRequester `optional:"true"`
	Events      agentports.AgentEventPublisher     `optional:"true"`
	Audit       auditports.AuditService
	Realtime    realtimeports.RealtimeService `optional:"true"`
	Logger      *zap.Logger
}

type LedgerService struct {
	db          kernelports.Transactor
	submissions factoringsubmissionports.SubmissionRepository
	entries     factoringledgerports.ReserveEntryRepository
	agreements  factoringports.AgreementRepository
	invoices    invoiceports.InvoiceRepository
	ledger      customerledgerports.CustomerLedgerProjectionRepository
	controls    tenantports.AccountingControlRepository
	writer      journalposting.Writer
	requester   factoringports.EvaluationRequester
	events      agentports.AgentEventPublisher
	rec         recorder
	now         func() int64
}

var _ factoringledgerports.LedgerService = (*LedgerService)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewLedgerService(p LedgerParams) factoringledgerports.LedgerService {
	l := p.Logger.Named("service.factoring-ledger")
	return &LedgerService{
		db:          p.DB,
		submissions: p.Submissions,
		entries:     p.Entries,
		agreements:  p.Agreements,
		invoices:    p.Invoices,
		ledger:      p.Ledger,
		controls:    p.Controls,
		writer: journalposting.Writer{
			Periods:  p.Periods,
			Numbers:  p.Sequences,
			Journals: p.Journals,
		},
		requester: p.Requester,
		events:    p.Events,
		rec:       recorder{audit: p.Audit, realtime: p.Realtime, l: l},
		now:       timeutils.NowUnix,
	}
}

func (s *LedgerService) ListEntries(
	ctx context.Context,
	req *factoringledgerports.ListReserveEntryConnectionRequest,
) (*pagination.CursorListResult[*factoring.ReserveEntry], error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	return s.entries.ListConnection(ctx, req)
}

func (s *LedgerService) GetBalance(
	ctx context.Context,
	req factoringledgerports.GetReserveBalanceRequest,
) (*factoringledgerports.ReserveBalance, error) {
	if req.FactoringCompanyID.IsNil() {
		return nil, errortypes.NewValidationError(
			"factoringCompanyId",
			errortypes.ErrRequired,
			"Choose the factor whose reserve to read",
		)
	}
	return s.entries.GetBalance(ctx, req)
}

type ledgerScope struct {
	before    *factoring.Submission
	agreement *factoring.Agreement
	control   *accountingcontrol.AccountingControl
	accounts  factoring.PostingAccounts
}

func (s *LedgerService) loadScope(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	submissionID pulid.ID,
	version int64,
) (*ledgerScope, error) {
	if submissionID.IsNil() {
		return nil, errortypes.NewValidationError(
			"submissionId",
			errortypes.ErrRequired,
			"Choose the schedule",
		)
	}
	sub, err := s.submissions.GetByID(ctx, factoringsubmissionports.GetSubmissionRequest{
		ID:           submissionID,
		TenantInfo:   tenantInfo,
		IncludeItems: true,
	})
	if err != nil {
		return nil, err
	}
	if sub.Version != version {
		return nil, errortypes.NewConflictError(
			"Schedule {0} changed since you opened it; reload and try again",
			sub.Number,
		)
	}
	agreement, err := s.agreements.GetByID(ctx, factoringports.GetAgreementRequest{
		ID:         sub.AgreementID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	control, err := s.controls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return nil, err
	}
	if control == nil {
		return nil, errortypes.NewBusinessError(
			"Set up accounting before booking factoring to the ledger",
		)
	}
	return &ledgerScope{
		before:    sub,
		agreement: agreement,
		control:   control,
		accounts: agreement.PostingAccounts(
			control.DefaultARAccountID,
			control.DefaultCashAccountID,
		),
	}, nil
}

func requireAccounts(
	accounts *factoring.PostingAccounts,
	treatment factoring.AccountingTreatment,
) error {
	missing := accounts.Missing(treatment)
	if len(missing) == 0 {
		return nil
	}
	multiErr := errortypes.NewMultiError()
	for _, m := range missing {
		multiErr.Add(m.Field, errortypes.ErrRequired, m.Message)
	}
	return multiErr
}

func requireExecutor(actor *permissionports.RequestActor, action string) error {
	if err := requireActor(actor, action); err != nil {
		return err
	}
	if actor.ExecutorUserID().IsNil() {
		return errortypes.NewAuthorizationError(
			"Factoring entries are booked to the ledger, so a person has to make or approve them",
		)
	}
	return nil
}

func (s *LedgerService) requireMovementDate(field string, at int64) error {
	switch {
	case at <= 0:
		return errortypes.NewValidationError(field, errortypes.ErrRequired, "Give the date")
	case at > s.now()+maxFutureMovementSeconds:
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"The date cannot be in the future",
		)
	default:
		return nil
	}
}

func (s *LedgerService) finish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	actor *permissionports.RequestActor,
	before, updated *factoring.Submission,
	comment string,
	invoiceIDs []pulid.ID,
) {
	kernelports.AfterCommit(ctx, func(committedCtx context.Context) {
		s.rec.log(actor, &auditEntry{
			resource: submissionResource,
			entity:   updated,
			previous: before,
			op:       permission.OpUpdate,
			comment:  comment,
		})
		s.rec.publish(committedCtx, submissionResource, updated, actor, submissionRealtimeEvent)
		if s.requester != nil && len(invoiceIDs) > 0 {
			s.requester.RequestEvaluation(
				committedCtx,
				tenantInfo,
				&factoringports.EvaluationTargets{
					InvoiceIDs: invoiceIDs,
				},
			)
		}
	})
}

func (s *LedgerService) publishEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	kind agent.EventKind,
	subjectID pulid.ID,
	related []pulid.ID,
	detail string,
) {
	agentports.PublishAgentEvent(ctx, s.events, agentports.AgentEvent{
		Kind:       kind,
		SubjectID:  subjectID,
		TenantInfo: tenantInfo,
		Related:    related,
		Detail:     detail,
	})
}
