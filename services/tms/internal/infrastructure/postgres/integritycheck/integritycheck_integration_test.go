//go:build integration

package integritycheck_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/integritycheck"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy/detentionpolicydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	policyTable = "detention_policies"
	day         = int64(24 * 60 * 60)
)

type policyShape func(*detentionpolicy.DetentionPolicy)

func seedPolicies(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	shapes map[string]policyShape,
) map[string]pulid.ID {
	t.Helper()

	scoped := tenant.Ctx(ctx)
	charge := &accessorialcharge.AccessorialCharge{
		Code:        "DETENTION",
		Description: "Detention",
		Method:      accessorialcharge.MethodFlat,
		Amount:      decimal.NewFromInt(75),
	}
	require.NoError(t, accessorialchargedb.AccessorialCharges.Create(scoped, db, charge))

	ids := make(map[string]pulid.ID, len(shapes))
	for code, shape := range shapes {
		policy := &detentionpolicy.DetentionPolicy{
			Name:                code + " policy",
			Code:                code,
			AccessorialChargeID: charge.ID,
		}
		require.NoError(t, detentionpolicydb.DetentionPolicies.Create(scoped, db, policy))
		policy.Status = detentionpolicy.PolicyStatusActive
		shape(policy)
		require.NoError(t, detentionpolicydb.DetentionPolicies.Save(scoped, db, policy))
		ids[code] = policy.ID
	}

	return ids
}

func unshaped(*detentionpolicy.DetentionPolicy) {}

func startsAt(at int64) policyShape {
	return func(p *detentionpolicy.DetentionPolicy) { p.EffectiveStartDate = &at }
}

func endsAt(at int64) policyShape {
	return func(p *detentionpolicy.DetentionPolicy) { p.EffectiveEndDate = &at }
}

func references(
	ids []pulid.ID,
	conditions ...validationframework.Condition,
) []validationframework.ReferenceCheck {
	checks := make([]validationframework.ReferenceCheck, 0, len(ids))
	for _, id := range ids {
		checks = append(checks, validationframework.ReferenceCheck{
			Table:      policyTable,
			ID:         id,
			Conditions: conditions,
		})
	}

	return checks
}

func checkReferences(
	t *testing.T,
	ctx context.Context,
	checker validationframework.IntegrityChecker,
	tenant *seedtest.TestData,
	checks []validationframework.ReferenceCheck,
) []bool {
	t.Helper()

	result, err := checker.CheckIntegrity(tenant.Ctx(ctx), &validationframework.IntegrityRequest{
		Table:          policyTable,
		OrganizationID: tenant.Organization.ID,
		BusinessUnitID: tenant.BusinessUnit.ID,
		References:     checks,
	})
	require.NoError(t, err)
	require.Len(t, result.Exists, len(checks))

	return result.Exists
}

func TestCheckIntegrityConditionOps(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenant := seedtest.SeedFullTestData(t, ctx, db)
	foreignTenant := seedtest.SeedAdditionalTenant(t, ctx, db, "IC")
	checker := integritycheck.New(postgres.WrapDB(db))

	now := timeutils.NowUnix()
	customerID := pulid.MustNew("cus_")
	ids := seedPolicies(t, ctx, db, tenant, map[string]policyShape{
		"PLAIN":   unshaped,
		"DRAFT":   func(p *detentionpolicy.DetentionPolicy) { p.Status = detentionpolicy.PolicyStatusDraft },
		"PAST":    startsAt(now - 30*day),
		"FUTURE":  startsAt(now + 30*day),
		"ENDING":  endsAt(now + 30*day),
		"ENDED":   endsAt(now - day),
		"APPT":    func(p *detentionpolicy.DetentionPolicy) { p.AppointmentStopsOnly = true },
		"SCOPED":  func(p *detentionpolicy.DetentionPolicy) { p.CustomerID = &customerID },
		"LAPSING": func(p *detentionpolicy.DetentionPolicy) { startsAt(now - day)(p); endsAt(now + day)(p) },
	})
	foreign := seedPolicies(t, ctx, db, foreignTenant, map[string]policyShape{"FOREIGN": unshaped})
	missing := pulid.MustNew("dtp_")

	t.Run("equality", func(t *testing.T) {
		exists := checkReferences(t, ctx, checker, tenant, references(
			[]pulid.ID{ids["PLAIN"], ids["DRAFT"], foreign["FOREIGN"], missing},
			validationframework.Condition{
				Column: "status",
				Value:  string(detentionpolicy.PolicyStatusActive),
			},
		))
		assert.Equal(t, []bool{true, false, false, false}, exists)

		exists = checkReferences(t, ctx, checker, tenant, references(
			[]pulid.ID{ids["PLAIN"], ids["APPT"]},
			validationframework.Condition{Column: "appointment_stops_only", Value: false},
		))
		assert.Equal(t, []bool{true, false}, exists)
	})

	t.Run("is null", func(t *testing.T) {
		exists := checkReferences(t, ctx, checker, tenant, references(
			[]pulid.ID{ids["PLAIN"], ids["ENDING"], ids["ENDED"], ids["PAST"]},
			validationframework.Condition{
				Column: "effective_end_date",
				Op:     validationframework.ConditionIsNull,
			},
		))
		assert.Equal(t, []bool{true, false, false, true}, exists)
	})

	t.Run("null or not after now", func(t *testing.T) {
		exists := checkReferences(t, ctx, checker, tenant, references(
			[]pulid.ID{ids["PLAIN"], ids["PAST"], ids["FUTURE"]},
			validationframework.Condition{
				Column: "effective_start_date",
				Op:     validationframework.ConditionNullOrNotAfterNow,
			},
		))
		assert.Equal(
			t,
			[]bool{true, true, false},
			exists,
			"null and past start pass, a future start does not",
		)
	})

	t.Run("ops on one column are separate groups", func(t *testing.T) {
		isNull := references(
			[]pulid.ID{ids["PAST"]},
			validationframework.Condition{
				Column: "effective_start_date",
				Op:     validationframework.ConditionIsNull,
			},
		)
		begun := references(
			[]pulid.ID{ids["PAST"]},
			validationframework.Condition{
				Column: "effective_start_date",
				Op:     validationframework.ConditionNullOrNotAfterNow,
			},
		)
		exists := checkReferences(t, ctx, checker, tenant, append(isNull, begun...))
		assert.Equal(t, []bool{false, true}, exists)
	})

	t.Run("combined conditions", func(t *testing.T) {
		exists := checkReferences(t, ctx, checker, tenant, references(
			[]pulid.ID{
				ids["PLAIN"],
				ids["PAST"],
				ids["DRAFT"],
				ids["FUTURE"],
				ids["ENDING"],
				ids["APPT"],
				ids["SCOPED"],
				ids["LAPSING"],
				foreign["FOREIGN"],
				missing,
				ids["PLAIN"],
			},
			validationframework.Condition{
				Column: "status",
				Value:  string(detentionpolicy.PolicyStatusActive),
			},
			validationframework.Condition{Column: "specificity_score", Value: 0},
			validationframework.Condition{Column: "appointment_stops_only", Value: false},
			validationframework.Condition{
				Column: "effective_end_date",
				Op:     validationframework.ConditionIsNull,
			},
			validationframework.Condition{
				Column: "effective_start_date",
				Op:     validationframework.ConditionNullOrNotAfterNow,
			},
		))
		assert.Equal(
			t,
			[]bool{true, true, false, false, false, false, false, false, false, false, true},
			exists,
		)
	})

	t.Run("unknown op is an error", func(t *testing.T) {
		_, err := checker.CheckIntegrity(tenant.Ctx(ctx), &validationframework.IntegrityRequest{
			Table: policyTable,
			References: references(
				[]pulid.ID{ids["PLAIN"]},
				validationframework.Condition{Column: "status", Op: "between"},
			),
		})
		require.Error(t, err)
	})
}
