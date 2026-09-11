package development

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	ptoPolicyStandardCode = "STD-DRIVER"
	ptoPolicyOwnerOpCode  = "OWNER-OP"
	ptoSeedDay            = int64(86400)
)

type PTOPolicySeed struct {
	seedhelpers.BaseSeed
}

// PTOPolicySeed gives the development org a working time-off book: a default
// accrual policy every seeded driver is enrolled in, an informational policy
// for owner-operators, opening balances, and the accruals the nightly job
// would have posted since each driver's hire date.
//
// Depends on:
//   - Worker: the drivers being enrolled
//   - DriverPay: owner-operator classification for the informational policy
func NewPTOPolicySeed() *PTOPolicySeed {
	seed := &PTOPolicySeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"PTOPolicy",
		"1.0.0",
		"Creates PTO accrual policies, enrols seeded workers, and posts opening balances and accruals",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedWorker, seedhelpers.SeedDriverPay)
	return seed
}

type ptoSeedRefs struct {
	orgID    pulid.ID
	buID     pulid.ID
	adminID  pulid.ID
	timezone *time.Location
}

func (s *PTOPolicySeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}
			loc, err := time.LoadLocation(org.Timezone)
			if err != nil {
				loc = time.UTC
			}
			refs := &ptoSeedRefs{
				orgID:    org.ID,
				buID:     org.BusinessUnitID,
				adminID:  admin.ID,
				timezone: loc,
			}

			standard, err := s.ensurePolicy(ctx, tx, refs, standardPTOPolicy(refs))
			if err != nil {
				return fmt.Errorf("ensure standard PTO policy: %w", err)
			}
			if _, err = s.ensurePolicy(ctx, tx, refs, ownerOperatorPTOPolicy(refs)); err != nil {
				return fmt.Errorf("ensure owner-operator PTO policy: %w", err)
			}

			workers, err := s.loadWorkers(ctx, tx, refs)
			if err != nil {
				return fmt.Errorf("load workers: %w", err)
			}

			for _, wrk := range workers {
				if err = s.enrolWorker(ctx, tx, refs, standard, wrk); err != nil {
					return fmt.Errorf("enrol worker %s: %w", wrk.ID, err)
				}
			}

			return nil
		},
	)
}

func standardPTOPolicy(refs *ptoSeedRefs) *worker.PTOPolicy {
	return &worker.PTOPolicy{
		OrganizationID:    refs.orgID,
		BusinessUnitID:    refs.buID,
		Name:              "Standard Driver",
		Code:              ptoPolicyStandardCode,
		Description:       "Default accrual policy for company drivers",
		Status:            worker.PTOPolicyStatusActive,
		IsDefault:         true,
		YearBasis:         worker.PTOYearBasisCalendarYear,
		CountWeekends:     true,
		WaitingPeriodDays: 90,
		RequiresApproval:  true,
		EnforceBalance:    true,
		Rules: []*worker.PTOPolicyRule{
			{
				PTOType:             worker.PTOTypeVacation,
				AccrualMethod:       worker.PTOAccrualMethodMonthly,
				AccrualAmountDays:   decimal.RequireFromString("0.83"),
				MaxBalanceDays:      decimal.NewNullDecimal(decimal.NewFromInt(20)),
				CarryoverCapDays:    decimal.NewNullDecimal(decimal.NewFromInt(5)),
				CarryoverExpiryDays: 90,
				OnTermination:       worker.PTOTerminationPayOut,
				Tiers: []worker.PTOAccrualTier{
					{
						MinMonths:         24,
						AccrualAmountDays: decimal.RequireFromString("1.25"),
						MaxBalanceDays:    decimal.NewNullDecimal(decimal.NewFromInt(25)),
					},
					{
						MinMonths:         60,
						AccrualAmountDays: decimal.RequireFromString("1.67"),
						MaxBalanceDays:    decimal.NewNullDecimal(decimal.NewFromInt(30)),
					},
				},
			},
			{
				PTOType:           worker.PTOTypeSick,
				AccrualMethod:     worker.PTOAccrualMethodFixedAnnualGrant,
				AccrualAmountDays: decimal.NewFromInt(5),
				MaxBalanceDays:    decimal.NewNullDecimal(decimal.NewFromInt(10)),
				CarryoverCapDays:  decimal.NewNullDecimal(decimal.Zero),
			},
			{
				PTOType:           worker.PTOTypePersonal,
				AccrualMethod:     worker.PTOAccrualMethodFixedAnnualGrant,
				AccrualAmountDays: decimal.NewFromInt(2),
				CarryoverCapDays:  decimal.NewNullDecimal(decimal.Zero),
			},
		},
	}
}

func ownerOperatorPTOPolicy(refs *ptoSeedRefs) *worker.PTOPolicy {
	return &worker.PTOPolicy{
		OrganizationID:   refs.orgID,
		BusinessUnitID:   refs.buID,
		Name:             "Owner Operator",
		Code:             ptoPolicyOwnerOpCode,
		Description:      "Tracks time off for owner-operators without enforcing a balance",
		Status:           worker.PTOPolicyStatusActive,
		YearBasis:        worker.PTOYearBasisCalendarYear,
		CountWeekends:    true,
		EnforceBalance:   false,
		RequiresApproval: true,
		Rules: []*worker.PTOPolicyRule{
			{
				PTOType:       worker.PTOTypeVacation,
				AccrualMethod: worker.PTOAccrualMethodNone,
			},
		},
	}
}

func (s *PTOPolicySeed) ensurePolicy(
	ctx context.Context,
	tx bun.Tx,
	refs *ptoSeedRefs,
	policy *worker.PTOPolicy,
) (*worker.PTOPolicy, error) {
	existing := new(worker.PTOPolicy)
	err := tx.NewSelect().
		Model(existing).
		Relation("Rules").
		Where("ptop.organization_id = ?", refs.orgID).
		Where("ptop.business_unit_id = ?", refs.buID).
		Where("ptop.code = ?", policy.Code).
		Scan(ctx)
	if err == nil {
		return existing, s.syncRuleFollowOns(ctx, tx, existing, policy)
	}

	if _, err = tx.NewInsert().Model(policy).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	for i, rule := range policy.Rules {
		rule.PTOPolicyID = policy.ID
		rule.OrganizationID = refs.orgID
		rule.BusinessUnitID = refs.buID
		rule.SortOrder = int32(i) //nolint:gosec // rule counts are tiny
	}
	if len(policy.Rules) > 0 {
		if _, err = tx.NewInsert().Model(&policy.Rules).Returning("*").Exec(ctx); err != nil {
			return nil, err
		}
	}

	return policy, nil
}

func (s *PTOPolicySeed) syncRuleFollowOns(
	ctx context.Context,
	tx bun.Tx,
	existing *worker.PTOPolicy,
	desired *worker.PTOPolicy,
) error {
	for _, rule := range existing.Rules {
		want := desired.RuleFor(rule.PTOType)
		if want == nil || (len(rule.Tiers) > 0 && rule.OnTermination != "") {
			continue
		}
		rule.Tiers = want.Tiers
		rule.OnTermination = want.TerminationAction()
		if _, err := tx.NewUpdate().
			Model(rule).
			Column("tiers", "on_termination").
			WherePK().
			Exec(ctx); err != nil {
			return fmt.Errorf("sync rule %s: %w", rule.PTOType, err)
		}
	}
	return nil
}

func (s *PTOPolicySeed) loadWorkers(
	ctx context.Context,
	tx bun.Tx,
	refs *ptoSeedRefs,
) ([]*worker.Worker, error) {
	workers := make([]*worker.Worker, 0, 32)
	err := tx.NewSelect().
		Model(&workers).
		Relation("Profile").
		Where("wrk.organization_id = ?", refs.orgID).
		Where("wrk.business_unit_id = ?", refs.buID).
		Where("wrk.status = ?", domaintypes.StatusActive).
		Where("wrk.type = ?", worker.WorkerTypeEmployee).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return workers, nil
}

func (s *PTOPolicySeed) enrolWorker(
	ctx context.Context,
	tx bun.Tx,
	refs *ptoSeedRefs,
	policy *worker.PTOPolicy,
	wrk *worker.Worker,
) error {
	exists, err := tx.NewSelect().
		Model((*worker.WorkerPTOPolicyAssignment)(nil)).
		Where("wppa.organization_id = ?", refs.orgID).
		Where("wppa.business_unit_id = ?", refs.buID).
		Where("wppa.worker_id = ?", wrk.ID).
		Where("wppa.effective_to IS NULL").
		Exists(ctx)
	if err != nil || exists {
		return err
	}

	now := timeutils.NowUnix()
	hireDate := now - 400*ptoSeedDay
	var termination *int64
	if wrk.Profile != nil {
		if wrk.Profile.HireDate > 0 {
			hireDate = wrk.Profile.HireDate
		}
		termination = wrk.Profile.TerminationDate
	}

	assignment := &worker.WorkerPTOPolicyAssignment{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		WorkerID:       wrk.ID,
		PTOPolicyID:    policy.ID,
		EffectiveFrom:  hireDate,
		AssignedByID:   refs.adminID,
		Note:           "Seeded default policy",
	}
	if _, err = tx.NewInsert().Model(assignment).Returning("*").Exec(ctx); err != nil {
		return err
	}

	opening := map[worker.PTOType]decimal.Decimal{
		worker.PTOTypeVacation: decimal.RequireFromString("4.5"),
		worker.PTOTypeSick:     decimal.NewFromInt(2),
	}

	for _, rule := range policy.Rules {
		bal := &worker.WorkerPTOBalance{
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			WorkerID:       wrk.ID,
			PTOType:        rule.PTOType,
		}
		if _, err = tx.NewInsert().Model(bal).Returning("*").Exec(ctx); err != nil {
			return err
		}

		entries := make([]*worker.WorkerPTOLedgerEntry, 0, 16)
		if days, ok := opening[rule.PTOType]; ok {
			entry := &worker.WorkerPTOLedgerEntry{
				EntryType:    worker.PTOLedgerEntryOpeningBalance,
				AmountDays:   days,
				EffectiveAt:  hireDate,
				PeriodKey:    "OB:" + assignment.ID.String(),
				AssignmentID: assignment.ID,
				PTOPolicyID:  policy.ID,
				ActorType:    worker.PTOLedgerActorUser,
				CreatedByID:  refs.adminID,
			}
			bal.Apply(entry)
			entries = append(entries, entry)
		}

		plan := ptoledgerservice.Schedule(ptoledgerservice.CalcInput{
			Rule:            *rule,
			YearBasis:       policy.YearBasis,
			WaitingDays:     policy.WaitingPeriodDays,
			HireDate:        hireDate,
			TerminationDate: termination,
			Loc:             refs.timezone,
			AsOf:            now,
			LookbackMonths:  12,
		})
		for _, planned := range plan {
			if planned.EntryType != worker.PTOLedgerEntryAccrual {
				continue
			}
			amount := planned.NominalDays
			if rule.MaxBalanceDays.Valid {
				room := rule.MaxBalanceDays.Decimal.Sub(bal.BalanceDays)
				if room.LessThan(amount) {
					amount = room
				}
			}
			bal.LastAccrualPeriodKey = planned.PeriodKey
			if !amount.IsPositive() {
				continue
			}
			entry := &worker.WorkerPTOLedgerEntry{
				EntryType:    worker.PTOLedgerEntryAccrual,
				AmountDays:   amount,
				EffectiveAt:  planned.EffectiveAt,
				PeriodKey:    planned.PeriodKey,
				AssignmentID: assignment.ID,
				PTOPolicyID:  policy.ID,
				ActorType:    worker.PTOLedgerActorSystem,
			}
			bal.Apply(entry)
			entries = append(entries, entry)
		}

		for _, entry := range entries {
			entry.OrganizationID = refs.orgID
			entry.BusinessUnitID = refs.buID
			entry.WorkerID = wrk.ID
			entry.PTOType = rule.PTOType
		}
		if len(entries) > 0 {
			if _, err = tx.NewInsert().Model(&entries).Returning("*").Exec(ctx); err != nil {
				return err
			}
		}
		if _, err = tx.NewUpdate().Model(bal).WherePK().Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
