package employment_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func activeState() employment.EmploymentState {
	return employment.EmploymentState{
		Status:        domaintypes.StatusActive,
		CanBeAssigned: true,
		HireDate:      1_700_000_000,
	}
}

func TestEmploymentEventKind_CanRecord(t *testing.T) {
	inactive := employment.EmploymentState{Status: domaintypes.StatusInactive, HireDate: 1}
	onLeave := activeState()
	onLeave.OpenLeave = true
	suspended := activeState()
	suspended.OpenSuspension = true

	tests := []struct {
		name  string
		kind  employment.EmploymentEventKind
		state employment.EmploymentState
		ok    bool
	}{
		{
			"hire a brand new worker",
			employment.EmploymentEventHired,
			employment.EmploymentState{Status: domaintypes.StatusActive},
			true,
		},
		{"cannot hire twice", employment.EmploymentEventHired, activeState(), false},
		{"rehire only when inactive", employment.EmploymentEventRehired, inactive, true},
		{"cannot rehire an active worker", employment.EmploymentEventRehired, activeState(), false},
		{"terminate an active worker", employment.EmploymentEventTerminated, activeState(), true},
		{"cannot terminate twice", employment.EmploymentEventTerminated, inactive, false},
		{"suspend", employment.EmploymentEventSuspended, activeState(), true},
		{"cannot double suspend", employment.EmploymentEventSuspended, suspended, false},
		{
			"reinstate needs a suspension",
			employment.EmploymentEventReinstated,
			activeState(),
			false,
		},
		{"reinstate lifts a suspension", employment.EmploymentEventReinstated, suspended, true},
		{"leave", employment.EmploymentEventLeaveStarted, activeState(), true},
		{"cannot stack leave", employment.EmploymentEventLeaveStarted, onLeave, false},
		{"leave end needs open leave", employment.EmploymentEventLeaveEnded, activeState(), false},
		{"leave end closes leave", employment.EmploymentEventLeaveEnded, onLeave, true},
		{"promote needs employment", employment.EmploymentEventPromoted, inactive, false},
		{"transfer", employment.EmploymentEventTransferred, activeState(), true},
		{"rate change", employment.EmploymentEventRateChanged, activeState(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.kind.CanRecord(tt.state)
			if tt.ok {
				assert.NoError(t, err)
				return
			}
			var verr *errortypes.Error
			require.ErrorAs(t, err, &verr)
			assert.Equal(t, "kind", verr.Field)
		})
	}
}

func TestEmploymentStateOf_TracksOpenLeaveAndSuspension(t *testing.T) {
	wrk := &worker.Worker{Status: domaintypes.StatusActive, CanBeAssigned: true}
	history := []*employment.WorkerEmploymentEvent{
		{Kind: employment.EmploymentEventHired},
		{Kind: employment.EmploymentEventSuspended},
		{Kind: employment.EmploymentEventReinstated},
		{Kind: employment.EmploymentEventLeaveStarted},
	}
	state := employment.EmploymentStateOf(wrk, history)
	assert.True(t, state.OpenLeave)
	assert.False(t, state.OpenSuspension)

	history = append(
		history,
		&employment.WorkerEmploymentEvent{Kind: employment.EmploymentEventTerminated},
	)
	state = employment.EmploymentStateOf(wrk, history)
	assert.False(t, state.OpenLeave, "termination closes any open leave")
}

func TestWorkerEmploymentEvent_Apply(t *testing.T) {
	newWorker := func() *worker.Worker {
		return &worker.Worker{
			Status:               domaintypes.StatusActive,
			CanBeAssigned:        true,
			AvailableForDispatch: true,
			DriverType:           worker.DriverTypeLocal,
			Type:                 worker.WorkerTypeEmployee,
			Profile:              &worker.WorkerProfile{HireDate: 1_600_000_000},
		}
	}

	t.Run("terminate closes employment and dispatch", func(t *testing.T) {
		wrk := newWorker()
		event := &employment.WorkerEmploymentEvent{
			Kind:        employment.EmploymentEventTerminated,
			EffectiveAt: 1_800_000_000,
		}
		assert.True(t, event.Apply(wrk))
		assert.Equal(t, domaintypes.StatusInactive, wrk.Status)
		assert.False(t, wrk.CanBeAssigned)
		assert.False(t, wrk.AvailableForDispatch)
		require.NotNil(t, wrk.Profile.TerminationDate)
		assert.Equal(t, int64(1_800_000_000), *wrk.Profile.TerminationDate)
	})

	t.Run("rehire reopens with a fresh hire date", func(t *testing.T) {
		wrk := newWorker()
		wrk.Status = domaintypes.StatusInactive
		term := int64(1_700_000_000)
		wrk.Profile.TerminationDate = &term
		event := &employment.WorkerEmploymentEvent{
			Kind:        employment.EmploymentEventRehired,
			EffectiveAt: 1_750_000_000,
		}
		assert.True(t, event.Apply(wrk))
		assert.Equal(t, domaintypes.StatusActive, wrk.Status)
		assert.True(t, wrk.CanBeAssigned)
		assert.Equal(t, int64(1_750_000_000), wrk.Profile.HireDate)
		assert.Nil(t, wrk.Profile.TerminationDate)
	})

	t.Run("suspend and reinstate toggle dispatch but not status", func(t *testing.T) {
		wrk := newWorker()
		assert.True(
			t,
			(&employment.WorkerEmploymentEvent{Kind: employment.EmploymentEventSuspended}).Apply(
				wrk,
			),
		)
		assert.Equal(t, domaintypes.StatusActive, wrk.Status)
		assert.False(t, wrk.CanBeAssigned)
		assert.True(
			t,
			(&employment.WorkerEmploymentEvent{Kind: employment.EmploymentEventReinstated}).Apply(
				wrk,
			),
		)
		assert.True(t, wrk.CanBeAssigned)
	})

	t.Run("transfer moves fleet code and manager from the recorded values", func(t *testing.T) {
		wrk := newWorker()
		event := &employment.WorkerEmploymentEvent{
			Kind: employment.EmploymentEventTransferred,
			ToValues: map[string]string{
				employment.EmploymentValueFleetCodeID: "fc_2",
				employment.EmploymentValueManagerID:   "usr_9",
			},
		}
		assert.True(t, event.Apply(wrk))
		assert.Equal(t, "fc_2", wrk.FleetCodeID.String())
		assert.Equal(t, "usr_9", wrk.ManagerID.String())
	})

	t.Run("promotion changes driver and worker type when valid", func(t *testing.T) {
		wrk := newWorker()
		event := &employment.WorkerEmploymentEvent{
			Kind: employment.EmploymentEventPromoted,
			ToValues: map[string]string{
				employment.EmploymentValueDriverType: "OTR",
				employment.EmploymentValueWorkerType: "Bogus",
			},
		}
		assert.True(t, event.Apply(wrk))
		assert.Equal(t, worker.DriverTypeOTR, wrk.DriverType)
		assert.Equal(t, worker.WorkerTypeEmployee, wrk.Type, "invalid values are ignored")
	})

	t.Run("informational kinds do not touch the worker", func(t *testing.T) {
		wrk := newWorker()
		assert.False(
			t,
			(&employment.WorkerEmploymentEvent{Kind: employment.EmploymentEventRateChanged}).Apply(
				wrk,
			),
		)
		assert.False(
			t,
			(&employment.WorkerEmploymentEvent{Kind: employment.EmploymentEventProbationEnded}).Apply(
				wrk,
			),
		)
	})
}

func TestWorkerEmploymentEvent_Validate(t *testing.T) {
	event := &employment.WorkerEmploymentEvent{Kind: employment.EmploymentEventTerminated}
	multiErr := errortypes.NewMultiError()
	event.Validate(multiErr)
	fields := map[string]bool{}
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}
	assert.True(t, fields["workerId"])
	assert.True(t, fields["effectiveAt"])
	assert.True(t, fields["reason"], "termination needs a reason")

	event.Kind = employment.EmploymentEventPromoted
	multiErr = errortypes.NewMultiError()
	event.Validate(multiErr)
	fields = map[string]bool{}
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}
	assert.False(t, fields["reason"], "promotions may omit the reason")
}

func TestTenure(t *testing.T) {
	const day = int64(86400)
	now := int64(1_800_000_000)
	assert.Equal(t, time.Duration(0), employment.Tenure(0, nil, now))
	assert.Equal(t, 10*24*time.Hour, employment.Tenure(now-10*day, nil, now))
	term := now - 4*day
	assert.Equal(t, 6*24*time.Hour, employment.Tenure(now-10*day, &term, now))
	future := now + 5*day
	assert.Equal(
		t,
		10*24*time.Hour,
		employment.Tenure(now-10*day, &future, now),
		"a future end date is ignored",
	)
	assert.Equal(t, time.Duration(0), employment.Tenure(now+day, nil, now))
}
