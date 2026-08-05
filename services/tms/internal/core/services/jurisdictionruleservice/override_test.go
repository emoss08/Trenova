package jurisdictionruleservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func fptr(v float64) *float64 { return &v }
func iptr(v int64) *int64     { return &v }
func i16ptr(v int16) *int16   { return &v }
func bptr(v bool) *bool       { return &v }

type overrideRepoStub struct {
	repositories.JurisdictionRuleRepository

	rule    *jurisdictionrule.JurisdictionRule
	created *jurisdictionrule.Override
}

func (s *overrideRepoStub) GetActiveByStateIDs(
	_ context.Context, _ *repositories.GetJurisdictionRulesRequest,
) ([]*jurisdictionrule.JurisdictionRule, error) {
	if s.rule == nil {
		return nil, nil
	}
	return []*jurisdictionrule.JurisdictionRule{s.rule}, nil
}

func (s *overrideRepoStub) CreateOverride(
	_ context.Context, e *jurisdictionrule.Override,
) (*jurisdictionrule.Override, error) {
	s.created = e
	return e, nil
}

func statuteRule() *jurisdictionrule.JurisdictionRule {
	return &jurisdictionrule.JurisdictionRule{
		StateID:            pulid.MustNew("us_"),
		MaxWidthFeet:       8.5,
		MaxHeightFeet:      13.5,
		MaxLengthFeet:      53,
		MaxWeightPounds:    80000,
		PermitLeadTimeDays: 2,
		PermitValidityDays: 5,
	}
}

func overrideFor(mutate func(*jurisdictionrule.Override)) *jurisdictionrule.Override {
	o := &jurisdictionrule.Override{
		StateID:        pulid.MustNew("us_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Reason:         "Our equipment and insurance posture is tighter than the statute",
	}
	mutate(o)
	return o
}

func overrideService(repo repositories.JurisdictionRuleRepository) *service {
	return &service{repo: repo, l: zap.NewNop()}
}

func fieldsOf(t *testing.T, err error) []string {
	t.Helper()

	multiErr, ok := err.(*errortypes.MultiError)
	require.True(t, ok, "expected a MultiError, got %T", err)

	fields := make([]string, 0, 4)
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	return fields
}

// Resolve already refuses to apply a looser override, so the save would not
// have taken effect — but it would have succeeded, leaving the operator
// believing a limit is in force that the engine ignores. These reject at write
// time so the answer is a field error rather than silence.
func TestCreateOverride_RejectsALooserLimit(t *testing.T) {
	cases := map[string]struct {
		mutate func(*jurisdictionrule.Override)
		field  string
	}{
		"width":    {func(o *jurisdictionrule.Override) { o.MaxWidthFeet = fptr(20) }, "maxWidthFeet"},
		"height":   {func(o *jurisdictionrule.Override) { o.MaxHeightFeet = fptr(18) }, "maxHeightFeet"},
		"length":   {func(o *jurisdictionrule.Override) { o.MaxLengthFeet = fptr(90) }, "maxLengthFeet"},
		"weight":   {func(o *jurisdictionrule.Override) { o.MaxWeightPounds = iptr(120000) }, "maxWeightPounds"},
		"leadTime": {func(o *jurisdictionrule.Override) { o.PermitLeadTimeDays = i16ptr(0) }, "permitLeadTimeDays"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc := overrideService(&overrideRepoStub{rule: statuteRule()})

			_, err := svc.CreateOverride(t.Context(), overrideFor(tc.mutate), nil)

			require.Error(t, err)
			assert.Contains(t, fieldsOf(t, err), tc.field)
		})
	}
}

func TestCreateOverride_AcceptsAStricterLimit(t *testing.T) {
	repo := &overrideRepoStub{rule: statuteRule()}
	svc := overrideService(repo)

	_, err := svc.CreateOverride(t.Context(), overrideFor(func(o *jurisdictionrule.Override) {
		o.MaxWidthFeet = fptr(8)
		o.MaxWeightPounds = iptr(70000)
		o.PermitLeadTimeDays = i16ptr(7)
	}), nil)

	require.NoError(t, err)
	require.NotNil(t, repo.created)
	assert.InDelta(t, 8, *repo.created.MaxWidthFeet, 0.001)
}

// A restriction the state imposes cannot be lifted by an override.
func TestCreateOverride_RejectsLiftingAStateRestriction(t *testing.T) {
	rule := statuteRule()
	rule.DaylightOnly = true
	rule.HolidayRestricted = true

	svc := overrideService(&overrideRepoStub{rule: rule})

	_, err := svc.CreateOverride(t.Context(), overrideFor(func(o *jurisdictionrule.Override) {
		o.DaylightOnly = bptr(false)
		o.HolidayRestricted = bptr(false)
	}), nil)

	require.Error(t, err)
	fields := fieldsOf(t, err)
	assert.Contains(t, fields, "daylightOnly")
	assert.Contains(t, fields, "holidayRestricted")
}

func TestCreateOverride_AllowsAddingARestrictionTheStateDoesNotImpose(t *testing.T) {
	svc := overrideService(&overrideRepoStub{rule: statuteRule()})

	_, err := svc.CreateOverride(t.Context(), overrideFor(func(o *jurisdictionrule.Override) {
		o.DaylightOnly = bptr(true)
	}), nil)

	require.NoError(t, err)
}

// With no statute on file there is nothing to compare against, and the engine
// still clamps. Failing here would block a carrier from recording a posture for
// a state nobody has configured yet.
func TestCreateOverride_AllowsAnOverrideWhereNoStatuteIsOnFile(t *testing.T) {
	svc := overrideService(&overrideRepoStub{rule: nil})

	_, err := svc.CreateOverride(t.Context(), overrideFor(func(o *jurisdictionrule.Override) {
		o.MaxWidthFeet = fptr(20)
	}), nil)

	require.NoError(t, err)
}

// The domain requires an override to change something; an empty one would be a
// reason attached to nothing.
func TestCreateOverride_RejectsAnOverrideThatChangesNothing(t *testing.T) {
	svc := overrideService(&overrideRepoStub{rule: statuteRule()})

	_, err := svc.CreateOverride(t.Context(), overrideFor(func(*jurisdictionrule.Override) {}), nil)

	require.Error(t, err)
}

func TestCreateOverride_RequiresAReason(t *testing.T) {
	svc := overrideService(&overrideRepoStub{rule: statuteRule()})

	_, err := svc.CreateOverride(t.Context(), overrideFor(func(o *jurisdictionrule.Override) {
		o.MaxWidthFeet = fptr(8)
		o.Reason = "short"
	}), nil)

	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "reason")
}
