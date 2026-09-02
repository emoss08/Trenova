package formulatemplate_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestNextRound_ContinuesWhileChangesAreRequested(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t, 1, formulatemplate.NextRound(nil), "the first submission opens round one")
	assert.EqualValues(t, 3, formulatemplate.NextRound(&formulatemplate.Review{
		Round: 2, Decision: formulatemplate.ReviewDecisionRejected,
	}), "a rejection closes the round")
	assert.EqualValues(t, 3, formulatemplate.NextRound(&formulatemplate.Review{
		Round: 2, Decision: formulatemplate.ReviewDecisionApproved,
	}))
	assert.EqualValues(t, 2, formulatemplate.NextRound(&formulatemplate.Review{
		Round: 2, Decision: formulatemplate.ReviewDecisionChangesRequested,
	}), "resubmitting after a change request stays in the same round")
	assert.EqualValues(t, 3, formulatemplate.NextRound(&formulatemplate.Review{
		Round: 2, Decision: formulatemplate.ReviewDecisionExpired,
	}), "an expired submission closes its round")
}

func TestSubmissionIsStale(t *testing.T) {
	t.Parallel()

	now := int64(1_800_000_000)
	fresh := now - 3*24*60*60
	old := now - formulatemplate.SubmissionExpiry - 1

	assert.False(t, formulatemplate.SubmissionIsStale(nil, now), "never submitted is not stale")
	assert.False(t, formulatemplate.SubmissionIsStale(&fresh, now))
	assert.True(t, formulatemplate.SubmissionIsStale(&old, now))
}

func TestReviewValidate_RequiresAnActorExceptForExpiry(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	cases := []struct {
		name    string
		review  formulatemplate.Review
		wantErr bool
	}{
		{
			name: "approval with actor",
			review: formulatemplate.Review{
				TemplateID: pulid.MustNew("ft_"), Round: 1,
				Decision: formulatemplate.ReviewDecisionApproved, ActorID: &actor,
			},
		},
		{
			name: "approval without actor",
			review: formulatemplate.Review{
				TemplateID: pulid.MustNew("ft_"), Round: 1,
				Decision: formulatemplate.ReviewDecisionApproved,
			},
			wantErr: true,
		},
		{
			name: "expiry without actor",
			review: formulatemplate.Review{
				TemplateID: pulid.MustNew("ft_"), Round: 1,
				Decision: formulatemplate.ReviewDecisionExpired,
			},
		},
		{
			name: "unknown decision",
			review: formulatemplate.Review{
				TemplateID: pulid.MustNew("ft_"), Round: 1,
				Decision: formulatemplate.ReviewDecision("Shrugged"), ActorID: &actor,
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			multiErr := errortypes.NewMultiError()
			tc.review.Validate(multiErr)
			assert.Equal(t, tc.wantErr, multiErr.HasErrors())
		})
	}
}
