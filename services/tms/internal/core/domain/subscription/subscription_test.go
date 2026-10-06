package subscription_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validSubscription() *subscription.Subscription {
	return &subscription.Subscription{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		PlanKey:        "free_demo",
		Status:         subscription.StatusTrialing,
		TrialEndsAt:    1_000,
		ReadOnlyUntil:  2_000,
	}
}

func TestSubscriptionValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*subscription.Subscription)
		field  string
	}{
		{name: "valid"},
		{name: "plan required", mutate: func(s *subscription.Subscription) { s.PlanKey = "" }, field: "planKey"},
		{name: "status invalid", mutate: func(s *subscription.Subscription) { s.Status = "paused" }, field: "status"},
		{name: "trial end required", mutate: func(s *subscription.Subscription) { s.TrialEndsAt = 0 }, field: "trialEndsAt"},
		{
			name:   "read only before trial end",
			mutate: func(s *subscription.Subscription) { s.ReadOnlyUntil = 999 },
			field:  "readOnlyUntil",
		},
		{
			name:   "organization required",
			mutate: func(s *subscription.Subscription) { s.OrganizationID = pulid.Nil },
			field:  "organizationId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sub := validSubscription()
			if tt.mutate != nil {
				tt.mutate(sub)
			}

			multiErr := errortypes.NewMultiError()
			sub.Validate(multiErr)

			if tt.field == "" {
				assert.False(t, multiErr.HasErrors(), multiErr.Error())
				return
			}

			assert.True(t, multiErr.HasErrors())
			fields := make([]string, 0, len(multiErr.Errors))
			for _, e := range multiErr.Errors {
				fields = append(fields, e.Field)
			}
			assert.Contains(t, fields, tt.field)
		})
	}
}

func TestSubscriptionEffectiveStatus(t *testing.T) {
	t.Parallel()

	sub := validSubscription()

	assert.Equal(t, subscription.StatusTrialing, sub.EffectiveStatus(999))
	assert.Equal(t, subscription.StatusReadOnly, sub.EffectiveStatus(1_000))
	assert.Equal(t, subscription.StatusExpired, sub.EffectiveStatus(2_000))

	sub.Status = subscription.StatusReadOnly
	assert.Equal(t, subscription.StatusReadOnly, sub.EffectiveStatus(1_999))
	assert.Equal(t, subscription.StatusExpired, sub.EffectiveStatus(2_000))

	sub.Status = subscription.StatusActive
	assert.Equal(t, subscription.StatusActive, sub.EffectiveStatus(5_000))

	sub.Status = subscription.StatusExpired
	assert.Equal(t, subscription.StatusExpired, sub.EffectiveStatus(0))
}

func TestStatusRules(t *testing.T) {
	t.Parallel()

	assert.True(t, subscription.StatusTrialing.AllowsWrites())
	assert.True(t, subscription.StatusActive.AllowsWrites())
	assert.False(t, subscription.StatusReadOnly.AllowsWrites())
	assert.False(t, subscription.StatusExpired.AllowsWrites())

	assert.True(t, subscription.StatusReadOnly.AllowsLogin())
	assert.False(t, subscription.StatusExpired.AllowsLogin())
	assert.False(t, subscription.Status("bogus").AllowsLogin())

	for _, status := range subscription.AllStatuses() {
		assert.True(t, status.IsValid())
	}
}
