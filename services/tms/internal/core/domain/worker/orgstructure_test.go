package worker_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validPosition() *worker.JobPosition {
	return &worker.JobPosition{
		ID:         "jpos_1",
		Status:     domaintypes.StatusActive,
		Code:       "DRV-OTR",
		Title:      "Over-the-Road Driver",
		Department: worker.DepartmentOperations,
	}
}

func TestJobPosition_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a complete position passes", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		validPosition().Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	// A position reporting to itself is a cycle of one, and the shortest way to
	// hang anything that walks the org chart.
	t.Run("a position cannot report to itself", func(t *testing.T) {
		t.Parallel()
		entity := validPosition()
		entity.ReportsToPositionID = entity.ID

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "cannot report to itself")
	})

	t.Run("a department outside the nine is refused", func(t *testing.T) {
		t.Parallel()
		entity := validPosition()
		entity.Department = worker.JobDepartment("Warehouse")

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "Department is not valid")
	})
}

// The unique index is on lower(code), so leaving the spelling to the caller
// would let "drv-otr" and "DRV-OTR" look different everywhere but the index.
func TestJobPosition_Normalise(t *testing.T) {
	t.Parallel()

	entity := &worker.JobPosition{
		Code:        "  drv-otr ",
		Title:       "  Over-the-Road Driver  ",
		Description: "  Long haul.  ",
	}
	entity.Normalise()

	assert.Equal(t, "DRV-OTR", entity.Code)
	assert.Equal(t, "Over-the-Road Driver", entity.Title)
	assert.Equal(t, "Long haul.", entity.Description)
}

func TestApprovalScope_Covers(t *testing.T) {
	t.Parallel()

	// All covers everything; anything else covers only itself. A delegation
	// that quietly covered more than it said would be worse than none.
	assert.True(t, worker.ApprovalScopeAll.Covers(worker.ApprovalScopeTimeOff))
	assert.True(t, worker.ApprovalScopeAll.Covers(worker.ApprovalScopeExpenses))
	assert.True(t, worker.ApprovalScopeTimeOff.Covers(worker.ApprovalScopeTimeOff))
	assert.False(t, worker.ApprovalScopeTimeOff.Covers(worker.ApprovalScopeExpenses))
	assert.False(t, worker.ApprovalScopeTimeOff.Covers(worker.ApprovalScopeAll))
}

func validDelegation() *worker.ApprovalDelegation {
	return &worker.ApprovalDelegation{
		DelegatorID: "usr_1",
		DelegateID:  "usr_2",
		Scope:       worker.ApprovalScopeAll,
		StartsAt:    1_800_000_000,
	}
}

func TestApprovalDelegation_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a complete delegation passes", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		validDelegation().Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	// Delegating to yourself reads on the list as cover somebody arranged,
	// which is worse than nothing while they are away.
	t.Run("a delegation has to hand approval to somebody else", func(t *testing.T) {
		t.Parallel()
		entity := validDelegation()
		entity.DelegateID = entity.DelegatorID

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "hand approval to somebody else")
	})

	t.Run("a delegation cannot end before it begins", func(t *testing.T) {
		t.Parallel()
		entity := validDelegation()
		before := entity.StartsAt - 1
		entity.EndsAt = &before

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "cannot end before it begins")
	})
}

func TestApprovalDelegation_IsActive(t *testing.T) {
	t.Parallel()

	start := int64(1_800_000_000)
	end := start + 7*86400

	t.Run("inside the window", func(t *testing.T) {
		t.Parallel()
		entity := validDelegation()
		entity.EndsAt = &end
		assert.True(t, entity.IsActive(start+86400))
	})

	t.Run("before it begins and after it ends", func(t *testing.T) {
		t.Parallel()
		entity := validDelegation()
		entity.EndsAt = &end
		assert.False(t, entity.IsActive(start-1))
		assert.False(t, entity.IsActive(end+1))
	})

	// An open end date runs until somebody revokes it, which is what a manager
	// handing over an area rather than a fortnight actually wants.
	t.Run("an open end date runs on", func(t *testing.T) {
		t.Parallel()
		entity := validDelegation()
		assert.True(t, entity.IsActive(start+365*86400))
	})

	// Revoking is what stops it, and the row is kept so an approval made under
	// it can still be explained.
	t.Run("revoking stops it from the moment it was revoked", func(t *testing.T) {
		t.Parallel()
		entity := validDelegation()
		revoked := start + 86400
		entity.RevokedAt = &revoked

		assert.True(t, entity.IsActive(start+3600), "still live before the revocation")
		assert.False(t, entity.IsActive(revoked))
		assert.False(t, entity.IsActive(revoked+3600))
	})
}
