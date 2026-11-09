package validationtest_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fieldpath"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recorder struct {
	testing.TB

	mu       sync.Mutex
	failures []string
	fatal    bool
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.mu.Lock()
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
	r.fatal = true
	r.mu.Unlock()

	runtime.Goexit()
}

func (r *recorder) FailNow() {
	r.mu.Lock()
	r.fatal = true
	r.mu.Unlock()

	runtime.Goexit()
}

func (r *recorder) failed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.failures) > 0
}

func (r *recorder) report() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return strings.Join(r.failures, "\n")
}

func record(t *testing.T, fn func(tb testing.TB)) *recorder {
	t.Helper()

	rec := &recorder{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(rec)
	}()
	<-done

	return rec
}

type stop struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Code           string
	LocationID     pulid.ID
}

func (s *stop) GetID() pulid.ID             { return s.ID }
func (s *stop) GetTableName() string        { return "stops" }
func (s *stop) GetOrganizationID() pulid.ID { return s.OrganizationID }
func (s *stop) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *stop) Validate(multiErr *errortypes.MultiError) {
	if s.Code == "" {
		multiErr.Add("code", errortypes.ErrRequired, "Code is required")
	}
}

func newStop() *stop {
	return &stop{
		ID:             pulid.MustNew("stp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Code:           "DAL",
		LocationID:     pulid.MustNew("loc_"),
	}
}

func refusal() error {
	multiErr := errortypes.NewMultiError()
	multiErr.WithIndex("stops", 1).Add("name", errortypes.ErrRequired, "Name is required")
	multiErr.Add(
		"code",
		errortypes.ErrInvalidLength,
		"Must be between {0} and {1} characters",
		1,
		5,
	)

	return multiErr.Err()
}

func failedRun() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("code", errortypes.ErrRequired, "Code is required")
	multiErr.AddInternal(errors.New("connection reset"))

	return multiErr.Err()
}

func TestRefusedReadsEachFieldErrorAtItsPath(t *testing.T) {
	t.Parallel()

	result := validationtest.Refused(t, refusal())

	got := result.Refuses("stops.1.name", errortypes.ErrRequired)
	assert.Equal(t, "Name is required", got.Message)
	result.Refuses("stops[1].name", errortypes.ErrRequired)
	assert.Equal(t, []string{"Must be between 1 and 5 characters"}, result.Messages("code"))
	assert.Equal(t, []errortypes.ErrorCode{errortypes.ErrInvalidLength}, result.Codes("code"))
	assert.ElementsMatch(t, []string{"stops[1].name", "code"}, result.Fields())
	assert.True(t, result.Exactly(
		validationtest.At("code", errortypes.ErrInvalidLength),
		validationtest.At("stops[1].name", errortypes.ErrRequired),
	))
	assert.True(t, result.Allows("stops[0]"))
}

func TestAssertionsFailWhenTheRefusalDiffers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		fatal bool
		run   func(tb testing.TB)
	}{
		{
			name:  "a code the field was not refused with",
			fatal: true,
			run: func(tb testing.TB) {
				validationtest.Refused(tb, refusal()).Refuses("code", errortypes.ErrRequired)
			},
		},
		{
			name: "a refusal the test did not name",
			run: func(tb testing.TB) {
				validationtest.Refused(tb, refusal()).
					Exactly(validationtest.At("code", errortypes.ErrInvalidLength))
			},
		},
		{
			name: "a refusal the test named that never happened",
			run: func(tb testing.TB) {
				validationtest.Refused(tb, refusal()).Exactly(
					validationtest.At("code", errortypes.ErrInvalidLength),
					validationtest.At("stops[1].name", errortypes.ErrRequired),
					validationtest.At("name", errortypes.ErrRequired),
				)
			},
		},
		{
			name: "a refusal nested under the field said to be allowed",
			run: func(tb testing.TB) {
				validationtest.Refused(tb, refusal()).Allows("stops")
			},
		},
		{
			name: "a refusal where none was wanted",
			run: func(tb testing.TB) {
				validationtest.Outcome(tb, refusal()).Valid()
			},
		},
		{
			name:  "no error where a refusal was wanted",
			fatal: true,
			run: func(tb testing.TB) {
				validationtest.Refused(tb, nil)
			},
		},
		{
			name:  "a run that could not complete read as a refusal",
			fatal: true,
			run: func(tb testing.TB) {
				validationtest.Outcome(tb, failedRun())
			},
		},
		{
			name:  "an error that is not a validation read as a refusal",
			fatal: true,
			run: func(tb testing.TB) {
				validationtest.Refused(tb, errors.New("not found"))
			},
		},
		{
			name:  "a refusal read as a run that could not complete",
			fatal: true,
			run: func(tb testing.TB) {
				validationtest.Failed(tb, refusal())
			},
		},
		{
			name:  "a refusal read as a version mismatch",
			fatal: true,
			run: func(tb testing.TB) {
				validationtest.VersionMismatch(tb, refusal())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := record(t, tt.run)

			require.True(t, rec.failed(), "the assertion passed")
			assert.Equal(t, tt.fatal, rec.fatal, rec.report())
		})
	}
}

func TestExactlyNamesWhatIsMissingAndWhatWasUnexpected(t *testing.T) {
	t.Parallel()

	rec := record(t, func(tb testing.TB) {
		validationtest.Refused(tb, refusal()).Exactly(
			validationtest.At("code", errortypes.ErrInvalidLength),
			validationtest.At("name", errortypes.ErrRequired),
		)
	})

	report := rec.report()
	assert.Contains(t, report, "missing:\n  - name REQUIRED")
	assert.Contains(t, report, "unexpected:\n  - stops[1].name REQUIRED: Name is required")
}

func TestFailedAndVersionMismatchReturnTheError(t *testing.T) {
	t.Parallel()

	err := validationtest.Failed(t, failedRun())
	assert.ErrorContains(t, err, "connection reset")

	conflict := validationtest.VersionMismatch(
		t,
		fmt.Errorf("update stop: %w", errortypes.NewVersionMismatchError("Stop changed")),
	)
	assert.NotNil(t, conflict)
}

func TestReportAndValidateReadTheWholeReport(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	child := multiErr.WithIndex("stops", 0)
	child.Add("code", errortypes.ErrRequired, "Code is required")
	validationtest.Report(t, child).Refuses("stops[0].code", errortypes.ErrRequired)

	validationtest.Report(t, nil).Valid()
	validationtest.Report(t, errortypes.NewMultiError()).Valid()

	entity := newStop()
	validationtest.Validate(t, entity).Valid()
	entity.Code = ""
	validationtest.Validate(t, entity).Exactly(validationtest.At("code", errortypes.ErrRequired))
}

func TestReportRefusesAReportThatCouldNotComplete(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	multiErr.AddInternal(errors.New("connection reset"))

	rec := record(t, func(tb testing.TB) {
		validationtest.Report(tb, multiErr)
	})

	assert.True(t, rec.fatal)
	assert.Contains(t, rec.report(), "connection reset")
}

func TestAdvisesFindsAnAdvisoryAtItsPath(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	multiErr.Add("code", errortypes.ErrRequired, "Code is required")
	multiErr.Advisories = append(multiErr.Advisories, &errortypes.AdvisoryError{
		Path:     fieldpath.Parse("stops[1].name"),
		Code:     errortypes.ErrInvalid,
		Message:  "Check the stop",
		Severity: errortypes.SeverityWarn,
	})

	result := validationtest.Report(t, multiErr)
	advisory := result.Advises("stops[1].name", errortypes.ErrInvalid)
	assert.Equal(t, errortypes.SeverityWarn, advisory.Severity)

	rec := record(t, func(tb testing.TB) {
		validationtest.Report(tb, multiErr).Advises("code", errortypes.ErrInvalid)
	})
	assert.True(t, rec.fatal)
}

func TestNewContextBuildsTheContextTheValidatorBuilds(t *testing.T) {
	t.Parallel()

	entity := newStop()
	created := validationtest.NewContext(t, entity)
	assert.True(t, created.IsCreate())
	assert.True(t, created.EntityID.IsNil())
	assert.Equal(t, entity.OrganizationID, created.OrganizationID)
	assert.Equal(t, entity.BusinessUnitID, created.BusinessUnitID)

	original := newStop()
	updated := validationtest.NewContext(t, entity, validationtest.UpdateFrom(original))
	assert.True(t, updated.IsUpdate())
	assert.Equal(t, entity.ID, updated.EntityID)
	got, ok := validationframework.OriginalOf[*stop](updated)
	require.True(t, ok)
	assert.Same(t, original, got)

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	other := validationtest.NewContext(t, entity, validationtest.InTenant(orgID, buID))
	assert.Equal(t, orgID, other.OrganizationID)
	assert.Equal(t, buID, other.BusinessUnitID)
}

func codeRule() *validationframework.BaseRule[*stop] {
	return validationframework.NewRule[*stop]("code_shape").
		WithValidation(func(
			_ context.Context,
			entity *stop,
			valCtx *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if len(entity.Code) != 3 {
				multiErr.Add("code", errortypes.ErrInvalidFormat, "Code must be three letters")
			}
			if valCtx.OrganizationID != entity.OrganizationID {
				multiErr.Add("organizationId", errortypes.ErrInvalid, "Wrong tenant")
			}

			return nil
		})
}

func TestRunRuleRunsOneRuleAsTheValidatorWould(t *testing.T) {
	t.Parallel()

	entity := newStop()
	validationtest.RunRule(t, codeRule(), entity).Valid()

	entity.Code = "DALLAS"
	validationtest.RunRule(t, codeRule(), entity).
		Exactly(validationtest.At("code", errortypes.ErrInvalidFormat))

	validationtest.RunRule(t, codeRule().OnCreate(), entity, validationtest.Update()).Valid()
}

func TestRunRuleFailureReturnsWhyTheRuleCouldNotComplete(t *testing.T) {
	t.Parallel()

	broken := validationframework.NewRule[*stop]("broken").
		WithValidation(func(
			context.Context,
			*stop,
			*validationframework.Context,
			*errortypes.MultiError,
		) error {
			return errors.New("dispatch control unavailable")
		})

	err := validationtest.RunRuleFailure(t, broken, newStop())
	assert.ErrorContains(t, err, "dispatch control unavailable")

	rec := record(t, func(tb testing.TB) {
		validationtest.RunRule(tb, broken, newStop())
	})
	assert.True(t, rec.fatal)

	rec = record(t, func(tb testing.TB) {
		bad := newStop()
		bad.Code = "DALLAS"
		validationtest.RunRuleFailure(tb, codeRule(), bad)
	})
	assert.True(t, rec.fatal, "a rule that refused did not fail to complete")
}

func TestIntegrityAnswersWhatTheTestDeclared(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("stp_")
	location := pulid.MustNew("loc_")
	integrity := validationtest.NewIntegrity().
		Taken(map[string]any{"code": "dal"}).
		TakenBy(owner, map[string]any{"code": "HOU"}).
		Taken(map[string]any{"code": "AUS", "location_id": location}).
		Missing("locations", location)

	pointed := "DAL"
	result, err := integrity.CheckIntegrity(t.Context(), &validationframework.IntegrityRequest{
		Table:     "stops",
		ExcludeID: owner,
		Unique: []validationframework.UniqueCheck{
			{Fields: []validationframework.FieldCheck{{Column: "code", Value: "DAL"}}},
			{Fields: []validationframework.FieldCheck{
				{Column: "code", Value: "DAL", CaseSensitive: true},
			}},
			{Fields: []validationframework.FieldCheck{{Column: "code", Value: "HOU"}}},
			{Fields: []validationframework.FieldCheck{{Column: "code", Value: &pointed}}},
			{Fields: []validationframework.FieldCheck{
				{Column: "code", Value: "AUS"},
				{Column: "location_id", Value: location},
			}},
			{Fields: []validationframework.FieldCheck{{Column: "code", Value: "AUS"}}},
		},
		References: []validationframework.ReferenceCheck{
			{Table: "locations", ID: location},
			{Table: "locations", ID: pulid.MustNew("loc_")},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, []bool{true, false, false, true, true, false}, result.Duplicate)
	assert.Equal(t, []bool{false, true}, result.Exists)
	require.Len(t, integrity.Requests(), 1)
	assert.Equal(t, owner, integrity.Requests()[0].ExcludeID)
}

func TestIntegrityRefusesWhatTheRealCheckerRefuses(t *testing.T) {
	t.Parallel()

	integrity := validationtest.NewIntegrity()
	unique := []validationframework.UniqueCheck{
		{Fields: []validationframework.FieldCheck{{Column: "code", Value: "DAL"}}},
	}

	_, err := integrity.CheckIntegrity(
		t.Context(),
		&validationframework.IntegrityRequest{Unique: unique},
	)
	require.Error(t, err)

	_, err = integrity.CheckIntegrity(t.Context(), &validationframework.IntegrityRequest{
		Table:  "stops",
		Unique: []validationframework.UniqueCheck{{}},
	})
	require.Error(t, err)

	failure := errors.New("database unavailable")
	_, err = validationtest.NewIntegrity().FailWith(failure).CheckIntegrity(
		t.Context(),
		&validationframework.IntegrityRequest{Table: "stops", Unique: unique},
	)
	require.ErrorIs(t, err, failure)
}

func stopValidator(integrity *validationtest.Integrity) *tenantedvalidation.Validator[*stop] {
	return tenantedvalidation.NewBuilder[*stop]().
		WithModelName("Stop").
		WithIntegrityChecker(integrity).
		WithUniqueField("code", "code", "", func(s *stop) any { return s.Code }).
		WithReferenceCheck("locationId", "locations", "", func(s *stop) pulid.ID {
			return s.LocationID
		}).
		Build()
}

func TestAValidatorRefusesWhatTheIntegrityFakeDeclares(t *testing.T) {
	t.Parallel()

	entity := newStop()
	entity.ID = pulid.Nil
	integrity := validationtest.NewIntegrity().
		Taken(map[string]any{"code": "dal"}).
		Missing("locations", entity.LocationID)

	result := validationtest.Refused(
		t,
		stopValidator(integrity).ValidateCreate(t.Context(), entity),
	)
	result.Exactly(
		validationtest.At("code", errortypes.ErrDuplicate),
		validationtest.At("locationId", errortypes.ErrInvalidReference),
	)
	assert.Equal(t, []string{"Stop with this value already exists"}, result.Messages("code"))

	requests := integrity.Requests()
	require.Len(t, requests, 1, "every check rides one batch")
	assert.Equal(t, entity.OrganizationID, requests[0].OrganizationID)
	assert.True(t, requests[0].ExcludeID.IsNil())
}

func TestAValidatorLetsARecordKeepItsOwnValue(t *testing.T) {
	t.Parallel()

	entity := newStop()
	integrity := validationtest.NewIntegrity().
		TakenBy(entity.ID, map[string]any{"code": entity.Code})

	validationtest.Outcome(t, stopValidator(integrity).ValidateUpdate(t.Context(), entity)).Valid()
	assert.Equal(t, entity.ID, integrity.Requests()[0].ExcludeID)
}

func TestAValidatorWhoseCheckerFailsCouldNotComplete(t *testing.T) {
	t.Parallel()

	integrity := validationtest.NewIntegrity().FailWith(errors.New("database unavailable"))

	err := validationtest.Failed(
		t,
		stopValidator(integrity).ValidateCreate(t.Context(), newStop()),
	)
	assert.ErrorContains(t, err, "database unavailable")
}
