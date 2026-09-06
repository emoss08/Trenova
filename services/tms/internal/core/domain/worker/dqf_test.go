package worker_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dqfDay = int64(86400)

func requiredCredential(
	code string,
	health worker.CredentialHealth,
) *worker.CredentialSummaryItem {
	return &worker.CredentialSummaryItem{
		CredentialType: &worker.WorkerCredentialType{Code: code, Name: code, IsRequired: true},
		Required:       true,
		Health:         health,
	}
}

func settledVerification() *worker.WorkerEmploymentVerification {
	received := int64(1_700_100_000)
	return &worker.WorkerEmploymentVerification{
		WorkerID:                      pulid.MustNew("wrk_"),
		EmployerName:                  "Prior Carrier",
		WasDOTRegulated:               true,
		Status:                        worker.VerificationReceived,
		Method:                        worker.VerificationByEmail,
		RequestedAt:                   ptrInt64(1_700_000_000),
		ResponseReceivedAt:            &received,
		DrugAlcoholResponseReceivedAt: &received,
	}
}

func clearStanding() *worker.DrugAlcoholStanding {
	return &worker.DrugAlcoholStanding{
		Status:                worker.DrugAlcoholClear,
		ReturnToDuty:          worker.ReturnToDutyNotRequired,
		HasPreEmploymentTest:  true,
		HasPreEmploymentQuery: true,
	}
}

func completeInput(now int64) worker.DQFInput {
	return worker.DQFInput{
		WorkerID: pulid.MustNew("wrk_"),
		HireDate: now - 400*dqfDay,
		Credentials: &worker.WorkerCredentialSummary{
			Items: []*worker.CredentialSummaryItem{
				requiredCredential("CDL", worker.CredentialHealthValid),
				requiredCredential("MED_CARD", worker.CredentialHealthValid),
			},
		},
		Verifications: []*worker.WorkerEmploymentVerification{settledVerification()},
		DrugAlcohol:   clearStanding(),
		Now:           now,
	}
}

func TestBuildDQF_CompleteFile(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	file := worker.BuildDQF(completeInput(now))

	assert.True(t, file.Complete)
	assert.Equal(t, 0, file.MissingRequired)
	assert.Equal(t, 0, file.Outstanding)
	assert.Nil(t, file.RetentionExpiresAt, "an employed driver's file has no purge date")
	assert.False(t, file.PurgeEligible)
}

// The file is assembled from the areas that own each piece. A missing required
// credential has to surface here, or a DQF audit would pass on a file the
// credential tab already says is incomplete.
func TestBuildDQF_MissingCredentialBlocks(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	in.Credentials.Items = append(
		in.Credentials.Items,
		requiredCredential("MVR", worker.CredentialHealthMissing),
	)

	file := worker.BuildDQF(in)

	assert.False(t, file.Complete)
	assert.Equal(t, 1, file.MissingRequired)
}

// A credential expiring next week is still valid today. Treating it as a gap
// would make every file incomplete for a month before every renewal.
func TestBuildDQF_ExpiringSoonWarnsWithoutBlocking(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	in.Credentials.Items = append(
		in.Credentials.Items,
		requiredCredential("ANNUAL_REVIEW", worker.CredentialHealthExpiringSoon),
	)

	file := worker.BuildDQF(in)

	assert.True(t, file.Complete)
	assert.Equal(t, 1, file.ExpiringSoon)
}

// Optional credentials a driver happens to hold are not part of the
// qualification file; listing them would pad an audit with cards nobody needs.
func TestBuildDQF_IgnoresOptionalCredentials(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	in.Credentials.Items = append(in.Credentials.Items, &worker.CredentialSummaryItem{
		CredentialType: &worker.WorkerCredentialType{Code: "TWIC", Name: "TWIC"},
		Required:       false,
		Health:         worker.CredentialHealthExpired,
	})

	file := worker.BuildDQF(in)

	assert.True(t, file.Complete)
	for _, item := range file.Items {
		assert.NotEqual(t, "TWIC", item.Code)
	}
}

// No employer recorded is not the same as none existing. Until somebody says
// the driver had no DOT-regulated employment, the investigation has not been
// made.
func TestBuildDQF_NoVerificationsIsMissing(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	in.Verifications = nil

	file := worker.BuildDQF(in)

	assert.False(t, file.Complete)
	assert.True(t, file.SafetyHistoryLate, "hired 400 days ago, so the 30-day window has passed")

	found := false
	for _, item := range file.Items {
		if item.Code == worker.DQFCodeSafetyHistory {
			found = true
			assert.Equal(t, worker.DQFMissing, item.Status)
		}
	}
	assert.True(t, found)
}

func TestBuildDQF_OutstandingVerificationBlocks(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	pending := settledVerification()
	pending.Status = worker.VerificationRequested
	pending.ResponseReceivedAt = nil
	in.Verifications = append(in.Verifications, pending)

	file := worker.BuildDQF(in)

	assert.False(t, file.Complete)
	assert.Equal(t, 1, file.Outstanding)
}

// An employer who never answers still settles the obligation: the rule asks for
// a good-faith effort and a record of it, not an answer nobody can compel.
func TestBuildDQF_NoResponseStillSettles(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	silent := settledVerification()
	silent.Status = worker.VerificationNoResponse
	silent.ResponseReceivedAt = nil
	silent.DrugAlcoholResponseReceivedAt = nil
	silent.FollowUpCount = 2
	in.Verifications = []*worker.WorkerEmploymentVerification{silent}

	file := worker.BuildDQF(in)

	assert.True(t, file.Complete)
}

// 49 CFR 382.413 asks for the drug and alcohol history specifically, and an
// employer often answers the general request without it.
func TestBuildDQF_ResponseWithoutDrugAlcoholHistoryIsOutstanding(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	in.Verifications[0].DrugAlcoholResponseReceivedAt = nil

	file := worker.BuildDQF(in)

	assert.False(t, file.Complete)
	assert.Equal(t, 1, file.Outstanding)
}

func TestBuildDQF_DrugAlcoholGates(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	in.DrugAlcohol = &worker.DrugAlcoholStanding{
		Status:       worker.DrugAlcoholUnknown,
		ReturnToDuty: worker.ReturnToDutyNotRequired,
	}

	file := worker.BuildDQF(in)

	assert.False(t, file.Complete)
	assert.Equal(t, 2, file.MissingRequired, "the test and the query are both gates")
}

// A prohibition belongs in the file even though it is not a missing document:
// an auditor reading the file has to see that the driver cannot be dispatched.
func TestBuildDQF_ProhibitionAppearsInTheFile(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	in.DrugAlcohol.Status = worker.DrugAlcoholProhibited
	in.DrugAlcohol.ReturnToDuty = worker.ReturnToDutyRTDTestRequired

	file := worker.BuildDQF(in)

	assert.False(t, file.Complete)
	found := false
	for _, item := range file.Items {
		if item.Code == worker.DQFCodeDrugAlcoholRecord {
			found = true
			assert.Equal(t, worker.DQFOutstanding, item.Status)
		}
	}
	assert.True(t, found)
}

func TestBuildDQF_Documents(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	in.Documents = &documentpacketrule.PacketSummary{
		Items: []documentpacketrule.PacketItemSummary{
			{
				DocumentTypeCode: "APPLICATION",
				DocumentTypeName: "Employment Application",
				Required:         true,
				Status:           documentpacketrule.ItemStatusMissing,
			},
			{
				DocumentTypeCode: "HANDBOOK",
				DocumentTypeName: "Handbook",
				Required:         false,
				Status:           documentpacketrule.ItemStatusMissing,
			},
		},
	}

	file := worker.BuildDQF(in)

	assert.False(t, file.Complete)
	assert.Equal(t, 1, file.MissingRequired, "only the required document counts")
}

// A tenant with no packet rules configured has no document section. Inventing
// gaps there would report a problem the organisation has not defined.
func TestBuildDQF_NoDocumentRulesContributesNothing(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	file := worker.BuildDQF(completeInput(now))

	for _, item := range file.Items {
		assert.NotEqual(t, worker.DQFSectionDocuments, item.Section)
	}
}

func TestBuildDQF_Retention(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	terminated := now - 1000*dqfDay
	in := completeInput(now)
	in.TerminationDate = &terminated

	file := worker.BuildDQF(in)
	require.NotNil(t, file.RetentionExpiresAt)
	assert.Equal(t, terminated+worker.DefaultDQFRetentionDays*dqfDay, *file.RetentionExpiresAt)
	assert.False(t, file.PurgeEligible, "1000 days is inside the three years")

	older := now - 1200*dqfDay
	in.TerminationDate = &older
	file = worker.BuildDQF(in)
	assert.True(t, file.PurgeEligible)

	// An organisation that chooses to hold longer holds longer.
	in.RetentionDays = 2000
	file = worker.BuildDQF(in)
	assert.False(t, file.PurgeEligible)
}

func TestBuildDQF_WorstFirstWithinASection(t *testing.T) {
	t.Parallel()

	now := int64(1_760_000_000)
	in := completeInput(now)
	in.Credentials.Items = []*worker.CredentialSummaryItem{
		requiredCredential("CDL", worker.CredentialHealthValid),
		requiredCredential("MED_CARD", worker.CredentialHealthMissing),
		requiredCredential("MVR", worker.CredentialHealthExpired),
	}

	file := worker.BuildDQF(in)

	require.GreaterOrEqual(t, len(file.Items), 3)
	assert.Equal(t, "MED_CARD", file.Items[0].Code)
	assert.Equal(t, "MVR", file.Items[1].Code)
}

func TestEmploymentVerification_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a requested investigation records when it was sent", func(t *testing.T) {
		t.Parallel()
		v := &worker.WorkerEmploymentVerification{
			WorkerID:     pulid.MustNew("wrk_"),
			EmployerName: "Prior Carrier",
			Status:       worker.VerificationRequested,
			Method:       worker.VerificationByEmail,
		}

		multiErr := errortypes.NewMultiError()
		v.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "when the request was sent")
	})

	t.Run("a received response records when it arrived", func(t *testing.T) {
		t.Parallel()
		v := &worker.WorkerEmploymentVerification{
			WorkerID:     pulid.MustNew("wrk_"),
			EmployerName: "Prior Carrier",
			Status:       worker.VerificationReceived,
			Method:       worker.VerificationByEmail,
			RequestedAt:  ptrInt64(1_700_000_000),
		}

		multiErr := errortypes.NewMultiError()
		v.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "when the response arrived")
	})

	t.Run("the response cannot pre-date the request", func(t *testing.T) {
		t.Parallel()
		v := settledVerification()
		v.ResponseReceivedAt = ptrInt64(1_699_000_000)

		multiErr := errortypes.NewMultiError()
		v.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "pre-date the request")
	})

	t.Run("reported accidents need a count", func(t *testing.T) {
		t.Parallel()
		v := settledVerification()
		v.HadAccidents = true

		multiErr := errortypes.NewMultiError()
		v.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "how many accidents")
	})

	t.Run("a settled investigation passes", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		settledVerification().Validate(multiErr)

		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})
}

// The drug and alcohol half only applies to DOT-regulated employment: time at a
// non-regulated employer has no testing record to ask about.
func TestEmploymentVerification_NeedsDrugAlcoholAnswer(t *testing.T) {
	t.Parallel()

	v := settledVerification()
	v.DrugAlcoholResponseReceivedAt = nil
	assert.True(t, v.NeedsDrugAlcoholAnswer())

	v.WasDOTRegulated = false
	assert.False(t, v.NeedsDrugAlcoholAnswer())

	v.WasDOTRegulated = true
	v.Status = worker.VerificationNotApplicable
	assert.False(t, v.NeedsDrugAlcoholAnswer())
}

func TestDueAtForHire(t *testing.T) {
	t.Parallel()

	hire := int64(1_700_000_000)
	assert.Equal(t, hire+worker.SafetyHistoryDueDays*dqfDay, worker.DueAtForHire(hire))
	assert.Equal(t, int64(0), worker.DueAtForHire(0))
}
