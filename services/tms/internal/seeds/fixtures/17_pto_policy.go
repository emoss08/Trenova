package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	ptoPolicyStandardCode = "STD-DRIVER"
	ptoPolicyOwnerOpCode  = "OWNER-OP"
	ptoSeedDay            = int64(86400)
)

type PTOPolicySeed struct {
}

// PTOPolicySeed gives the development org a working time-off book: a default
// accrual policy every seeded driver is enrolled in, an informational policy
// for owner-operators, opening balances, and the accruals the nightly job
// would have posted since each driver's hire date.
//
// Depends on:
//   - Worker: the drivers being enrolled
//   - DriverPay: owner-operator classification for the informational policy
var PTOPolicy = Seeds.Add(seed.Seed{
	Name:        "pto_policy",
	Kind:        seed.Fixture,
	Description: "Creates PTO accrual policies, enrols seeded workers, and posts opening balances and accruals",
	After:       []*seed.Seed{Worker, DriverPay},
	Run:         (&PTOPolicySeed{}).Run,
})

type ptoSeedRefs struct {
	orgID    pulid.ID
	buID     pulid.ID
	adminID  pulid.ID
	timezone *time.Location
}

func (s *PTOPolicySeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := userByUsername(ctx, db, "admin")
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

	standard, err := s.ensurePolicy(ctx, db, refs, standardPTOPolicy(refs))
	if err != nil {
		return fmt.Errorf("ensure standard PTO policy: %w", err)
	}
	if _, err = s.ensurePolicy(ctx, db, refs, ownerOperatorPTOPolicy(refs)); err != nil {
		return fmt.Errorf("ensure owner-operator PTO policy: %w", err)
	}

	workers, err := s.loadWorkers(ctx, db, refs)
	if err != nil {
		return fmt.Errorf("load workers: %w", err)
	}

	for _, wrk := range workers {
		if err = s.enrolWorker(ctx, db, refs, standard, wrk); err != nil {
			return fmt.Errorf("enrol worker %s: %w", wrk.ID, err)
		}
	}

	return nil
}

func standardPTOPolicy(refs *ptoSeedRefs) *pto.PTOPolicy {
	return &pto.PTOPolicy{
		OrganizationID:    refs.orgID,
		BusinessUnitID:    refs.buID,
		Name:              "Standard Driver",
		Code:              ptoPolicyStandardCode,
		Description:       "Default accrual policy for company drivers",
		Status:            pto.PTOPolicyStatusActive,
		IsDefault:         true,
		YearBasis:         pto.PTOYearBasisCalendarYear,
		CountWeekends:     true,
		WaitingPeriodDays: 90,
		RequiresApproval:  true,
		EnforceBalance:    true,
		Rules: []*pto.PTOPolicyRule{
			{
				PTOType:             worker.PTOTypeVacation,
				AccrualMethod:       pto.PTOAccrualMethodMonthly,
				AccrualAmountDays:   decimal.RequireFromString("0.83"),
				MaxBalanceDays:      decimal.NewNullDecimal(decimal.NewFromInt(20)),
				CarryoverCapDays:    decimal.NewNullDecimal(decimal.NewFromInt(5)),
				CarryoverExpiryDays: 90,
				OnTermination:       pto.PTOTerminationPayOut,
				Tiers: []pto.PTOAccrualTier{
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
				AccrualMethod:     pto.PTOAccrualMethodFixedAnnualGrant,
				AccrualAmountDays: decimal.NewFromInt(5),
				MaxBalanceDays:    decimal.NewNullDecimal(decimal.NewFromInt(10)),
				CarryoverCapDays:  decimal.NewNullDecimal(decimal.Zero),

				OnTermination: pto.PTOTerminationForfeit,

				Tiers: []pto.PTOAccrualTier{},
			},
			{
				PTOType:           worker.PTOTypePersonal,
				AccrualMethod:     pto.PTOAccrualMethodFixedAnnualGrant,
				AccrualAmountDays: decimal.NewFromInt(2),
				CarryoverCapDays:  decimal.NewNullDecimal(decimal.Zero),

				OnTermination: pto.PTOTerminationForfeit,

				Tiers: []pto.PTOAccrualTier{},
			},
		},
	}
}

func ownerOperatorPTOPolicy(refs *ptoSeedRefs) *pto.PTOPolicy {
	return &pto.PTOPolicy{
		OrganizationID:   refs.orgID,
		BusinessUnitID:   refs.buID,
		Name:             "Owner Operator",
		Code:             ptoPolicyOwnerOpCode,
		Description:      "Tracks time off for owner-operators without enforcing a balance",
		Status:           pto.PTOPolicyStatusActive,
		YearBasis:        pto.PTOYearBasisCalendarYear,
		CountWeekends:    true,
		EnforceBalance:   false,
		RequiresApproval: true,
		Rules: []*pto.PTOPolicyRule{
			{
				PTOType:       worker.PTOTypeVacation,
				AccrualMethod: pto.PTOAccrualMethodNone,

				OnTermination: pto.PTOTerminationForfeit,

				Tiers: []pto.PTOAccrualTier{},
			},
		},
	}
}

func (s *PTOPolicySeed) ensurePolicy(
	ctx context.Context,
	db *keel.DB,
	refs *ptoSeedRefs,
	policy *pto.PTOPolicy,
) (*pto.PTOPolicy, error) {
	existing, err := workerdb.PTOPolicies.AllTenants().Load(workerdb.PTOPolicies.Rules()).Where(
		workerdb.PTOPolicies.OrganizationID.Eq(refs.orgID),
		workerdb.PTOPolicies.BusinessUnitID.Eq(refs.buID),
		workerdb.PTOPolicies.Code.Eq(policy.Code),
	).First(ctx, db)
	if err == nil {
		return existing, s.syncRuleFollowOns(ctx, db, existing, policy)
	}

	if err = workerdb.PTOPolicies.Create(ctx, db, policy); err != nil {
		return nil, err
	}
	for i, rule := range policy.Rules {
		rule.PTOPolicyID = policy.ID
		rule.OrganizationID = refs.orgID
		rule.BusinessUnitID = refs.buID
		rule.SortOrder = int32(i) //nolint:gosec // rule counts are tiny
	}
	if len(policy.Rules) > 0 {
		if err = workerdb.PTOPolicyRules.CreateMany(ctx, db, policy.Rules); err != nil {
			return nil, err
		}
	}

	return policy, nil
}

func (s *PTOPolicySeed) syncRuleFollowOns(
	ctx context.Context,
	db *keel.DB,
	existing *pto.PTOPolicy,
	desired *pto.PTOPolicy,
) error {
	for _, rule := range existing.Rules {
		want := desired.RuleFor(rule.PTOType)
		if want == nil || (len(rule.Tiers) > 0 && rule.OnTermination != "") {
			continue
		}
		rule.Tiers = want.Tiers
		rule.OnTermination = want.TerminationAction()
		pr := workerdb.PTOPolicyRules
		if err := pr.Save(ctx, db, rule, pr.Tiers, pr.OnTermination); err != nil {
			return fmt.Errorf("sync rule %s: %w", rule.PTOType, err)
		}
	}
	return nil
}

func (s *PTOPolicySeed) loadWorkers(
	ctx context.Context,
	db *keel.DB,
	refs *ptoSeedRefs,
) ([]*worker.Worker, error) {
	workers, err := workerdb.Workers.AllTenants().Load(workerdb.Workers.Profile()).Where(
		workerdb.Workers.OrganizationID.Eq(refs.orgID),
		workerdb.Workers.BusinessUnitID.Eq(refs.buID),
		workerdb.Workers.Status.Eq(domaintypes.StatusActive),
		workerdb.Workers.Type.Eq(worker.WorkerTypeEmployee),
	).All(ctx, db)
	if err != nil {
		return nil, err
	}
	return workers, nil
}

func (s *PTOPolicySeed) enrolWorker(
	ctx context.Context,
	db *keel.DB,
	refs *ptoSeedRefs,
	policy *pto.PTOPolicy,
	wrk *worker.Worker,
) error {
	exists, err := workerdb.WorkerPTOPolicyAssignments.AllTenants().Where(
		workerdb.WorkerPTOPolicyAssignments.OrganizationID.Eq(refs.orgID),
		workerdb.WorkerPTOPolicyAssignments.BusinessUnitID.Eq(refs.buID),
		workerdb.WorkerPTOPolicyAssignments.WorkerID.Eq(wrk.ID),
		workerdb.WorkerPTOPolicyAssignments.EffectiveTo.IsNull(),
	).Exists(ctx, db)
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

	assignment := &pto.WorkerPTOPolicyAssignment{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		WorkerID:       wrk.ID,
		PTOPolicyID:    policy.ID,
		EffectiveFrom:  hireDate,
		AssignedByID:   refs.adminID,
		Note:           "Seeded default policy",
	}
	if err = workerdb.WorkerPTOPolicyAssignments.Create(ctx, db, assignment); err != nil {
		return err
	}

	opening := map[worker.PTOType]decimal.Decimal{
		worker.PTOTypeVacation: decimal.RequireFromString("4.5"),
		worker.PTOTypeSick:     decimal.NewFromInt(2),
	}

	for _, rule := range policy.Rules {
		bal := &pto.WorkerPTOBalance{
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			WorkerID:       wrk.ID,
			PTOType:        rule.PTOType,
		}
		if err = workerdb.WorkerPTOBalances.Create(ctx, db, bal); err != nil {
			return err
		}

		entries := make([]*pto.WorkerPTOLedgerEntry, 0, 16)
		if days, ok := opening[rule.PTOType]; ok {
			entry := &pto.WorkerPTOLedgerEntry{
				EntryType:    pto.PTOLedgerEntryOpeningBalance,
				AmountDays:   days,
				EffectiveAt:  hireDate,
				PeriodKey:    "OB:" + assignment.ID.String(),
				AssignmentID: assignment.ID,
				PTOPolicyID:  policy.ID,
				ActorType:    pto.PTOLedgerActorUser,
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
			if planned.Deferred {
				if amount, posts := ptoledgerservice.ResolveExpiry(bal, planned, rule); posts {
					entry := &pto.WorkerPTOLedgerEntry{
						EntryType:    planned.EntryType,
						AmountDays:   amount,
						EffectiveAt:  planned.EffectiveAt,
						PeriodKey:    planned.PeriodKey,
						AssignmentID: assignment.ID,
						PTOPolicyID:  policy.ID,
						ActorType:    pto.PTOLedgerActorSystem,
					}
					bal.Apply(entry)
					entries = append(entries, entry)
				}
				ptoledgerservice.AdvanceCursor(bal, planned, rule)
				continue
			}
			if planned.EntryType != pto.PTOLedgerEntryAccrual {
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
			entry := &pto.WorkerPTOLedgerEntry{
				EntryType:    pto.PTOLedgerEntryAccrual,
				AmountDays:   amount,
				EffectiveAt:  planned.EffectiveAt,
				PeriodKey:    planned.PeriodKey,
				AssignmentID: assignment.ID,
				PTOPolicyID:  policy.ID,
				ActorType:    pto.PTOLedgerActorSystem,
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
			if err = workerdb.WorkerPTOLedgerEntries.CreateMany(ctx, db, entries); err != nil {
				return err
			}
		}
		if err = workerdb.WorkerPTOBalances.Save(ctx, db, bal); err != nil {
			return err
		}
	}

	return nil
}
