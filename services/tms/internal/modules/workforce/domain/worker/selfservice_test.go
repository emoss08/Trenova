package worker_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpolicy"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func completePolicy() *workerpolicy.WorkerPolicy {
	return &workerpolicy.WorkerPolicy{
		Status:        domaintypes.StatusActive,
		Code:          "HANDBOOK",
		Title:         "Driver handbook",
		Body:          "Be safe.",
		VersionLabel:  "2026.1",
		AppliesTo:     workerpolicy.PolicyAudienceAll,
		EffectiveFrom: 1_800_000_000,
	}
}

func TestPolicyAudience_Covers(t *testing.T) {
	t.Parallel()

	assert.True(t, workerpolicy.PolicyAudienceAll.Covers(worker.WorkerTypeEmployee))
	assert.True(t, workerpolicy.PolicyAudienceAll.Covers(worker.WorkerTypeContractor))
	// A handbook for employees is not a contract term for an owner-operator.
	assert.True(t, workerpolicy.PolicyAudienceEmployees.Covers(worker.WorkerTypeEmployee))
	assert.False(t, workerpolicy.PolicyAudienceEmployees.Covers(worker.WorkerTypeContractor))
	assert.False(t, workerpolicy.PolicyAudienceContractors.Covers(worker.WorkerTypeEmployee))
}

func TestWorkerPolicy_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a complete policy passes", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		completePolicy().Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	// A policy that says nothing is a signature with nothing behind it.
	t.Run("a policy needs text or a document", func(t *testing.T) {
		t.Parallel()
		policy := completePolicy()
		policy.Body = "   "

		multiErr := errortypes.NewMultiError()
		policy.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "signature has to be on something")
	})

	t.Run("an attached document is enough", func(t *testing.T) {
		t.Parallel()
		policy := completePolicy()
		policy.Body = ""
		policy.DocumentID = "doc_1"

		multiErr := errortypes.NewMultiError()
		policy.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})
}

// The title and summary are not words anybody signed; the body and the
// document are.
func TestWorkerPolicy_ContentChanged(t *testing.T) {
	t.Parallel()

	before := completePolicy()
	after := completePolicy()
	after.Title = "Driver handbook (revised)"
	after.Summary = "Now with more."
	assert.False(t, after.ContentChanged(before))

	after.Body = "Be safer."
	assert.True(t, after.ContentChanged(before))

	after.Body = before.Body
	after.DocumentID = "doc_2"
	assert.True(t, after.ContentChanged(before))
}

func TestWorkerPolicyAcknowledgement_Signed(t *testing.T) {
	t.Parallel()

	ack := &workerpolicy.WorkerPolicyAcknowledgement{SignatureName: "  "}
	assert.False(t, ack.Signed())
	ack.SignatureName = "Ada Byron"
	assert.True(t, ack.Signed())
}

func TestDiffContact(t *testing.T) {
	t.Parallel()

	current := profilechange.ContactSnapshot{
		PhoneNumber:  "555-0100",
		AddressLine1: "1 Main St",
		City:         "Springfield",
		PostalCode:   "12345",
	}

	// Only the fields that would actually move are listed, in reading order,
	// so the person deciding sees exactly what is being asked.
	t.Run("lists only what moves", func(t *testing.T) {
		t.Parallel()
		wanted := current
		wanted.PostalCode = "54321"
		wanted.PhoneNumber = " 555-0100 "

		changes := profilechange.DiffContact(current, wanted)

		require.Len(t, changes, 1)
		assert.Equal(t, profilechange.ProfileFieldPostalCode, changes[0].Field)
		assert.Equal(t, "12345", changes[0].From)
		assert.Equal(t, "54321", changes[0].To)
	})

	t.Run("an unchanged snapshot is no request at all", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, profilechange.DiffContact(current, current))
	})

	t.Run("reads in address order", func(t *testing.T) {
		t.Parallel()
		wanted := current
		wanted.EmergencyContactPhone = "555-0199"
		wanted.PhoneNumber = "555-0101"
		wanted.City = "Shelbyville"

		changes := profilechange.DiffContact(current, wanted)

		require.Len(t, changes, 3)
		assert.Equal(t, profilechange.ProfileFieldPhoneNumber, changes[0].Field)
		assert.Equal(t, profilechange.ProfileFieldCity, changes[1].Field)
		assert.Equal(t, profilechange.ProfileFieldEmergencyContactPhone, changes[2].Field)
	})
}

func TestWorkerProfileChangeRequest_Validate(t *testing.T) {
	t.Parallel()

	base := func() *profilechange.WorkerProfileChangeRequest {
		return &profilechange.WorkerProfileChangeRequest{
			WorkerID:    "wrk_1",
			Status:      profilechange.ProfileChangePending,
			SubmittedAt: 1_800_000_000,
			Changes: []profilechange.FieldChange{
				{Field: profilechange.ProfileFieldCity, From: "A", To: "B"},
			},
		}
	}

	t.Run("a request that changes nothing is refused", func(t *testing.T) {
		t.Parallel()
		req := base()
		req.Changes = nil

		multiErr := errortypes.NewMultiError()
		req.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "Nothing would change")
	})

	// Compliance fields stay carrier-controlled: a driver who could edit their
	// own qualifications could also edit them into compliance.
	t.Run("only the closed list of fields can be asked for", func(t *testing.T) {
		t.Parallel()
		req := base()
		req.Changes = []profilechange.FieldChange{{Field: "licenseNumber", From: "A", To: "B"}}

		multiErr := errortypes.NewMultiError()
		req.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "not something you can change")
	})

	// A rejection with no reason leaves a driver with a record that did not
	// change and no idea why.
	t.Run("turning a request down needs a reason", func(t *testing.T) {
		t.Parallel()
		req := base()
		decided := int64(1_800_000_100)
		req.Status = profilechange.ProfileChangeRejected
		req.DecidedAt = &decided

		multiErr := errortypes.NewMultiError()
		req.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "needs a reason")
	})
}

// An approval applies exactly what was asked and nothing else — never a field
// outside the closed list, whatever the stored changes say.
func TestWorkerProfileChangeRequest_ApplyTo(t *testing.T) {
	t.Parallel()

	w := &worker.Worker{PhoneNumber: "old", City: "Springfield", AddressLine1: "1 Main St"}
	req := &profilechange.WorkerProfileChangeRequest{
		Changes: []profilechange.FieldChange{
			{Field: profilechange.ProfileFieldPhoneNumber, From: "old", To: "new"},
			{Field: "licenseNumber", From: "x", To: "y"},
		},
	}

	req.ApplyTo(w)

	assert.Equal(t, "new", w.PhoneNumber)
	assert.Equal(t, "Springfield", w.City)
	assert.Equal(t, "1 Main St", w.AddressLine1)
}

func TestProfileChangeStatus_CanTransitionTo(t *testing.T) {
	t.Parallel()

	assert.True(
		t,
		profilechange.ProfileChangePending.CanTransitionTo(profilechange.ProfileChangeApproved),
	)
	assert.True(
		t,
		profilechange.ProfileChangePending.CanTransitionTo(profilechange.ProfileChangeRejected),
	)
	assert.True(
		t,
		profilechange.ProfileChangePending.CanTransitionTo(profilechange.ProfileChangeWithdrawn),
	)
	// Every decision is final: an approval is already on the record, and a
	// rejection is asked for again rather than reopened.
	for _, terminal := range []profilechange.ProfileChangeStatus{
		profilechange.ProfileChangeApproved,
		profilechange.ProfileChangeRejected,
		profilechange.ProfileChangeWithdrawn,
	} {
		assert.False(
			t,
			terminal.CanTransitionTo(profilechange.ProfileChangePending),
			string(terminal),
		)
		assert.False(t, terminal.IsOpen(), string(terminal))
	}
}
