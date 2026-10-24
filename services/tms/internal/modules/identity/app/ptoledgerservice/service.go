package ptoledgerservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeResource = "worker_pto_balance"
	lockTimeout      = 5 * time.Second
)

var ErrCapped = errors.New("pto accrual capped at maximum balance")

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           kernelports.Transactor
	LedgerRepo   workerports.PTOLedgerRepository
	PolicyRepo   workerports.PTOPolicyRepository
	WorkerRepo   workerports.WorkerRepository
	OrgRepo      tenantports.OrganizationRepository
	Settlement   tenantports.SettlementControlRepository `optional:"true"`
	Holidays     workerports.OrgHolidayRepository        `optional:"true"`
	AuditService auditports.AuditService
	Realtime     agentports.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	db           kernelports.Transactor
	ledgerRepo   workerports.PTOLedgerRepository
	policyRepo   workerports.PTOPolicyRepository
	workerRepo   workerports.WorkerRepository
	orgRepo      tenantports.OrganizationRepository
	settlement   tenantports.SettlementControlRepository
	holidays     workerports.OrgHolidayRepository
	auditService auditports.AuditService
	realtime     agentports.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.pto-ledger"),
		db:           p.DB,
		ledgerRepo:   p.LedgerRepo,
		policyRepo:   p.PolicyRepo,
		workerRepo:   p.WorkerRepo,
		orgRepo:      p.OrgRepo,
		settlement:   p.Settlement,
		holidays:     p.Holidays,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

// PayPeriodSpec reads the organisation's settlement calendar for per-pay-period
// accrual; nil when settlement is not configured.
func (s *Service) PayPeriodSpec(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PayPeriodSpec, error) {
	if s.settlement == nil {
		return nil, nil //nolint:nilnil // no settlement calendar configured
	}
	control, err := s.settlement.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return &PayPeriodSpec{
		Frequency:    control.PayPeriodFrequency,
		EndDayOfWeek: control.PeriodEndDayOfWeek,
	}, nil
}

// HolidayCalendar loads the organisation's holidays and blackouts; an empty
// calendar when none are configured.
func (s *Service) HolidayCalendar(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.HolidayCalendar, error) {
	if s.holidays == nil {
		return worker.NewHolidayCalendar(nil), nil
	}
	rows, err := s.holidays.List(ctx, &workerports.ListOrgHolidaysRequest{TenantInfo: tenantInfo})
	if err != nil {
		return nil, err
	}
	return worker.NewHolidayCalendar(rows), nil
}

type Actor struct {
	Type   worker.PTOLedgerActorType
	UserID pulid.ID
}

func UserActor(userID pulid.ID) Actor {
	return Actor{Type: worker.PTOLedgerActorUser, UserID: userID}
}

func SystemActor() Actor {
	return Actor{Type: worker.PTOLedgerActorSystem}
}

type ResolvedPolicy struct {
	Assignment *worker.WorkerPTOPolicyAssignment
	Policy     *worker.PTOPolicy
}

func (r *ResolvedPolicy) RuleFor(ptoType worker.PTOType) *worker.PTOPolicyRule {
	if r == nil || r.Policy == nil {
		return nil
	}
	return r.Policy.RuleFor(ptoType)
}

func (s *Service) ResolvePolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	asOf int64,
) (*ResolvedPolicy, error) {
	assignment, err := s.policyRepo.GetActiveAssignment(ctx, &workerports.GetPTOAssignmentRequest{
		TenantInfo:    tenantInfo,
		WorkerID:      workerID,
		AsOf:          asOf,
		IncludePolicy: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	if assignment.Policy == nil || !assignment.Policy.IsActive() {
		return nil, nil
	}
	return &ResolvedPolicy{Assignment: assignment, Policy: assignment.Policy}, nil
}

func (s *Service) OrgLocation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*time.Location, error) {
	org, err := s.orgRepo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if org.Timezone == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(org.Timezone)
	if err != nil {
		s.l.Warn("invalid organization timezone; using UTC",
			zap.String("timezone", org.Timezone), zap.Error(err))
		return time.UTC, nil
	}
	return loc, nil
}

type postParams struct {
	key       workerports.PTOBalanceKey
	entry     *worker.WorkerPTOLedgerEntry
	actor     Actor
	rule      *worker.PTOPolicyRule
	resolve   func(bal *worker.WorkerPTOBalance) (decimal.Decimal, bool)
	onApplied func(bal *worker.WorkerPTOBalance)
}

func (s *Service) post(
	ctx context.Context,
	p postParams,
) (*worker.WorkerPTOLedgerEntry, error) {
	entry := p.entry
	var posted *worker.WorkerPTOLedgerEntry

	err := s.db.Transact(
		ctx,
		kernelports.TxOptions{LockTimeout: lockTimeout},
		func(txCtx context.Context) error {
			if err := s.ledgerRepo.EnsureBalance(txCtx, &p.key); err != nil {
				return err
			}
			bal, err := s.ledgerRepo.LockBalance(txCtx, &p.key)
			if err != nil {
				return err
			}

			amount := entry.AmountDays
			if p.resolve != nil {
				resolved, ok := p.resolve(bal)
				if !ok {
					return s.skipEntry(txCtx, bal, p)
				}
				amount = resolved
			}
			if entry.EntryType == worker.PTOLedgerEntryAccrual && p.rule != nil &&
				p.rule.MaxBalanceDays.Valid {
				room := p.rule.MaxBalanceDays.Decimal.Sub(bal.BalanceDays)
				if room.LessThan(amount) {
					amount = room
				}
			}
			if amount.IsZero() || (entry.EntryType.RequiredSign() > 0 && !amount.IsPositive()) ||
				(entry.EntryType.RequiredSign() < 0 && !amount.IsNegative()) {
				return s.skipEntry(txCtx, bal, p)
			}

			entry.AmountDays = amount
			entry.OrganizationID = p.key.TenantInfo.OrgID
			entry.BusinessUnitID = p.key.TenantInfo.BuID
			entry.WorkerID = p.key.WorkerID
			entry.PTOType = p.key.PTOType
			entry.ActorType = p.actor.Type
			entry.CreatedByID = p.actor.UserID
			if entry.EffectiveAt == 0 {
				entry.EffectiveAt = time.Now().Unix()
			}
			bal.Apply(entry)

			multiErr := errortypes.NewMultiError()
			entry.Validate(multiErr)
			if multiErr.HasErrors() {
				return multiErr
			}

			if _, err = s.ledgerRepo.InsertEntry(txCtx, entry); err != nil {
				return err
			}
			if p.onApplied != nil {
				p.onApplied(bal)
			}
			if _, err = s.ledgerRepo.UpdateBalance(txCtx, bal); err != nil {
				return err
			}
			posted = entry
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	s.publish(ctx, p.key, p.actor)
	return posted, nil
}

func (s *Service) skipEntry(
	ctx context.Context,
	bal *worker.WorkerPTOBalance,
	p postParams,
) error {
	if p.onApplied != nil {
		p.onApplied(bal)
		if _, err := s.ledgerRepo.UpdateBalance(ctx, bal); err != nil {
			return err
		}
	}
	return ErrCapped
}

func (s *Service) publish(ctx context.Context, key workerports.PTOBalanceKey, actor Actor) {
	if s.realtime == nil {
		return
	}
	actorType := permissionports.PrincipalTypeUser
	actorID := actor.UserID
	if actor.Type == worker.PTOLedgerActorSystem {
		actorType = permissionports.PrincipalTypeSystem
		actorID = permissionports.SystemPrincipalID
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: key.TenantInfo.OrgID,
		BusinessUnitID: key.TenantInfo.BuID,
		ActorUserID:    actor.UserID,
		ActorType:      actorType,
		ActorID:        actorID,
		Resource:       realtimeResource,
		Action:         "updated",
		RecordID:       key.WorkerID,
	}); err != nil {
		s.l.Warn("failed to publish PTO balance invalidation", zap.Error(err))
	}
}

func (s *Service) PostUsage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pto *worker.WorkerPTO,
	actor Actor,
) (*worker.WorkerPTOLedgerEntry, error) {
	if !pto.Days.IsPositive() {
		return nil, nil
	}
	resolved, err := s.ResolvePolicy(ctx, tenantInfo, pto.WorkerID, pto.StartDate)
	if err != nil {
		return nil, err
	}
	if resolved.RuleFor(pto.Type) == nil {
		return nil, nil
	}

	entry, err := s.post(ctx, postParams{
		key: workerports.PTOBalanceKey{
			TenantInfo: tenantInfo,
			WorkerID:   pto.WorkerID,
			PTOType:    pto.Type,
		},
		actor: actor,
		entry: &worker.WorkerPTOLedgerEntry{
			EntryType:    worker.PTOLedgerEntryUsage,
			AmountDays:   pto.Days.Neg(),
			EffectiveAt:  pto.StartDate,
			SourcePTOID:  pto.ID,
			AssignmentID: resolved.Assignment.ID,
			PTOPolicyID:  resolved.Policy.ID,
		},
	})
	if errors.Is(err, workerports.ErrDuplicatePTOLedgerEntry) {
		return nil, nil
	}
	return entry, err
}

func (s *Service) PostReversal(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pto *worker.WorkerPTO,
	actor Actor,
) (*worker.WorkerPTOLedgerEntry, error) {
	hasUsage, err := s.ledgerRepo.HasEntry(ctx, &workerports.HasPTOLedgerEntryRequest{
		TenantInfo:  tenantInfo,
		SourcePTOID: pto.ID,
		EntryType:   worker.PTOLedgerEntryUsage,
	})
	if err != nil {
		return nil, err
	}
	if !hasUsage || !pto.Days.IsPositive() {
		return nil, nil
	}

	entry, err := s.post(ctx, postParams{
		key: workerports.PTOBalanceKey{
			TenantInfo: tenantInfo,
			WorkerID:   pto.WorkerID,
			PTOType:    pto.Type,
		},
		actor: actor,
		entry: &worker.WorkerPTOLedgerEntry{
			EntryType:   worker.PTOLedgerEntryReversal,
			AmountDays:  pto.Days,
			EffectiveAt: time.Now().Unix(),
			SourcePTOID: pto.ID,
		},
	})
	if errors.Is(err, workerports.ErrDuplicatePTOLedgerEntry) {
		return nil, nil
	}
	return entry, err
}

type AdjustRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerID    pulid.ID
	PTOType     worker.PTOType
	AmountDays  decimal.Decimal
	EffectiveAt int64
	Note        string
	UserID      pulid.ID
}

func (s *Service) Adjust(
	ctx context.Context,
	req *AdjustRequest,
) (*worker.WorkerPTOLedgerEntry, error) {
	if err := validateAdjust(req); err != nil {
		return nil, err
	}

	entry, err := s.post(ctx, postParams{
		key: workerports.PTOBalanceKey{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			PTOType:    req.PTOType,
		},
		actor: UserActor(req.UserID),
		entry: &worker.WorkerPTOLedgerEntry{
			EntryType:   worker.PTOLedgerEntryAdjustment,
			AmountDays:  req.AmountDays,
			EffectiveAt: req.EffectiveAt,
			Note:        req.Note,
		},
	})
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceWorkerPTO,
		ResourceID:     req.WorkerID.String(),
		Operation:      permission.OpManage,
		UserID:         req.UserID,
		CurrentState:   jsonutils.MustToJSON(entry),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}, auditservice.WithComment(fmt.Sprintf("PTO balance adjusted: %s", req.Note))); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return entry, nil
}

type OpeningBalance struct {
	PTOType worker.PTOType
	Days    decimal.Decimal
}

func (s *Service) PostOpeningBalances(
	ctx context.Context,
	assignment *worker.WorkerPTOPolicyAssignment,
	balances []OpeningBalance,
	actor Actor,
) error {
	tenantInfo := pagination.TenantInfo{
		OrgID: assignment.OrganizationID,
		BuID:  assignment.BusinessUnitID,
	}
	for _, ob := range balances {
		if !ob.Days.IsPositive() {
			continue
		}
		_, err := s.post(ctx, postParams{
			key: workerports.PTOBalanceKey{
				TenantInfo: tenantInfo,
				WorkerID:   assignment.WorkerID,
				PTOType:    ob.PTOType,
			},
			actor: actor,
			entry: &worker.WorkerPTOLedgerEntry{
				EntryType:    worker.PTOLedgerEntryOpeningBalance,
				AmountDays:   ob.Days,
				EffectiveAt:  assignment.EffectiveFrom,
				PeriodKey:    "OB:" + assignment.ID.String(),
				AssignmentID: assignment.ID,
				PTOPolicyID:  assignment.PTOPolicyID,
			},
		})
		if err != nil && !errors.Is(err, workerports.ErrDuplicatePTOLedgerEntry) {
			return err
		}
	}
	return nil
}

func (s *Service) EnsureBalances(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	policy *worker.PTOPolicy,
) error {
	for _, rule := range policy.Rules {
		if rule == nil {
			continue
		}
		if err := s.ledgerRepo.EnsureBalance(ctx, &workerports.PTOBalanceKey{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			PTOType:    rule.PTOType,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ListLedger(
	ctx context.Context,
	req *workerports.ListPTOLedgerRequest,
) (*pagination.CursorListResult[*worker.WorkerPTOLedgerEntry], error) {
	return s.ledgerRepo.ListEntries(ctx, req)
}

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*workerports.PTOBalanceSummary, error) {
	return s.ledgerRepo.Summary(ctx, tenantInfo)
}

func (s *Service) RebuildBalance(
	ctx context.Context,
	key *workerports.PTOBalanceKey,
) (*worker.WorkerPTOBalance, error) {
	var rebuilt *worker.WorkerPTOBalance
	err := s.db.Transact(
		ctx,
		kernelports.TxOptions{LockTimeout: lockTimeout},
		func(txCtx context.Context) error {
			if err := s.ledgerRepo.EnsureBalance(txCtx, key); err != nil {
				return err
			}
			bal, err := s.ledgerRepo.LockBalance(txCtx, key)
			if err != nil {
				return err
			}
			total, count, err := s.ledgerRepo.SumEntries(txCtx, key)
			if err != nil {
				return err
			}
			bal.BalanceDays = total
			bal.EntryCount = count
			rebuilt, err = s.ledgerRepo.UpdateBalance(txCtx, bal)
			return err
		},
	)
	if err != nil {
		return nil, err
	}
	return rebuilt, nil
}
