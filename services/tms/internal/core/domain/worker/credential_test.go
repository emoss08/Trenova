package worker_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(86400)

func ptr(v int64) *int64 { return &v }

func TestEvaluateCredentialHealth(t *testing.T) {
	now := int64(1_800_000_000)
	now -= now % day
	now += 3600

	tests := []struct {
		name   string
		expiry *int64
		window int32
		want   worker.CredentialHealth
	}{
		{name: "no expiry never expires", expiry: nil, window: 30, want: worker.CredentialHealthValid},
		{name: "zero expiry treated as unset", expiry: ptr(0), window: 30, want: worker.CredentialHealthValid},
		{name: "far future", expiry: ptr(now + 90*day), window: 30, want: worker.CredentialHealthValid},
		{name: "one day outside window", expiry: ptr(now + 31*day), window: 30, want: worker.CredentialHealthValid},
		{name: "on window boundary", expiry: ptr(now + 30*day), window: 30, want: worker.CredentialHealthExpiringSoon},
		{name: "expires today", expiry: ptr(now - 3600 + 1), window: 30, want: worker.CredentialHealthExpiringSoon},
		{name: "expired yesterday", expiry: ptr(now - day), window: 30, want: worker.CredentialHealthExpired},
		{name: "zero window only warns on the day", expiry: ptr(now + 1*day), window: 0, want: worker.CredentialHealthValid},
		{name: "zero window day of", expiry: ptr(now), window: 0, want: worker.CredentialHealthExpiringSoon},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := worker.EvaluateCredentialHealth(tt.expiry, tt.window, now)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDaysUntil(t *testing.T) {
	now := int64(1_800_000_000)
	now -= now % day
	assert.Equal(t, int64(0), worker.DaysUntil(now+day-1, now+3600))
	assert.Equal(t, int64(1), worker.DaysUntil(now+day, now+3600))
	assert.Equal(t, int64(-1), worker.DaysUntil(now-1, now))
}

func credType(code string, required bool, driverTypes ...worker.DriverType) *worker.WorkerCredentialType {
	return &worker.WorkerCredentialType{
		ID:                     pulid.MustNew("wct_"),
		Code:                   code,
		Name:                   code,
		Status:                 domaintypes.StatusActive,
		IsRequired:             required,
		RequiredForDriverTypes: driverTypes,
		RenewalWindowDays:      30,
	}
}

func TestWorkerCredentialType_AppliesTo(t *testing.T) {
	otr := &worker.Worker{DriverType: worker.DriverTypeOTR}
	local := &worker.Worker{DriverType: worker.DriverTypeLocal}

	everyone := credType("A", true)
	otrOnly := credType("B", true, worker.DriverTypeOTR, worker.DriverTypeRegional)
	optional := credType("C", false)
	inactive := credType("D", true)
	inactive.Status = domaintypes.StatusInactive

	assert.True(t, everyone.AppliesTo(otr))
	assert.True(t, everyone.AppliesTo(local))
	assert.True(t, otrOnly.AppliesTo(otr))
	assert.False(t, otrOnly.AppliesTo(local))
	assert.False(t, optional.AppliesTo(otr))
	assert.False(t, inactive.AppliesTo(otr))
	assert.False(t, otrOnly.AppliesTo(nil))
}

func TestBuildCredentialSummary(t *testing.T) {
	now := int64(1_800_000_000)
	wrk := &worker.Worker{ID: pulid.MustNew("wrk_"), DriverType: worker.DriverTypeOTR}

	cdl := credType("CDL", true)
	cdl.SortOrder = 10
	med := credType("MED_CARD", true)
	med.SortOrder = 20
	twic := credType("TWIC", false)
	twic.SortOrder = 60
	forklift := credType("FORKLIFT", false)
	localOnly := credType("LOCAL", true, worker.DriverTypeLocal)

	active := func(typ *worker.WorkerCredentialType, expiry *int64) *worker.WorkerCredential {
		return &worker.WorkerCredential{
			ID:               pulid.MustNew("wcred_"),
			WorkerID:         wrk.ID,
			CredentialTypeID: typ.ID,
			Status:           worker.CredentialStatusActive,
			ExpiresAt:        expiry,
		}
	}
	archived := active(med, ptr(now+400*day))
	archived.Status = worker.CredentialStatusArchived

	t.Run("required slots always appear and roll up to compliance", func(t *testing.T) {
		summary := worker.BuildCredentialSummary(
			wrk,
			[]*worker.WorkerCredentialType{forklift, twic, med, cdl, localOnly},
			[]*worker.WorkerCredential{
				active(cdl, ptr(now+10*day)),
				archived,
				active(twic, ptr(now+200*day)),
			},
			now,
		)

		require.Len(t, summary.Items, 3)
		assert.Equal(t, "CDL", summary.Items[0].CredentialType.Code)
		assert.Equal(t, worker.CredentialHealthExpiringSoon, summary.Items[0].Health)
		assert.Equal(t, int64(10), *summary.Items[0].DaysUntilExpiry)
		assert.Equal(t, "MED_CARD", summary.Items[1].CredentialType.Code)
		assert.Equal(t, worker.CredentialHealthMissing, summary.Items[1].Health)
		assert.Nil(t, summary.Items[1].Credential)
		assert.Equal(t, "TWIC", summary.Items[2].CredentialType.Code)
		assert.False(t, summary.Items[2].Required)

		assert.Equal(t, 2, summary.RequiredCount)
		assert.Equal(t, 1, summary.ValidCount)
		assert.Equal(t, 1, summary.ExpiringCount)
		assert.Equal(t, 1, summary.MissingCount)
		assert.Equal(t, worker.ComplianceStatusNonCompliant, summary.ComplianceStatus)

		attention := summary.Attention()
		require.Len(t, attention, 2)
		assert.Equal(t, worker.CredentialHealthMissing, attention[0].Health)
		assert.Equal(t, worker.CredentialHealthExpiringSoon, attention[1].Health)
	})

	t.Run("expiring soon warns without blocking", func(t *testing.T) {
		summary := worker.BuildCredentialSummary(
			wrk,
			[]*worker.WorkerCredentialType{cdl, med},
			[]*worker.WorkerCredential{
				active(cdl, ptr(now+5*day)),
				active(med, nil),
			},
			now,
		)
		assert.Equal(t, worker.ComplianceStatusCompliant, summary.ComplianceStatus)
		assert.Equal(t, 1, summary.ExpiringCount)
		assert.Equal(t, 1, summary.ValidCount)
	})

	t.Run("expired required credential blocks", func(t *testing.T) {
		summary := worker.BuildCredentialSummary(
			wrk,
			[]*worker.WorkerCredentialType{cdl},
			[]*worker.WorkerCredential{active(cdl, ptr(now-day))},
			now,
		)
		assert.Equal(t, worker.ComplianceStatusNonCompliant, summary.ComplianceStatus)
		assert.Equal(t, 1, summary.ExpiredCount)
	})

	t.Run("optional expired credential does not block", func(t *testing.T) {
		summary := worker.BuildCredentialSummary(
			wrk,
			[]*worker.WorkerCredentialType{twic},
			[]*worker.WorkerCredential{active(twic, ptr(now-day))},
			now,
		)
		assert.Equal(t, worker.ComplianceStatusCompliant, summary.ComplianceStatus)
		assert.Equal(t, 1, summary.ExpiredCount)
	})

	t.Run("credentials of types outside the catalog still show when the relation is loaded", func(t *testing.T) {
		orphan := credType("LEGACY", false)
		cred := active(orphan, nil)
		cred.CredentialType = orphan
		summary := worker.BuildCredentialSummary(wrk, nil, []*worker.WorkerCredential{cred}, now)
		require.Len(t, summary.Items, 1)
		assert.Equal(t, worker.CredentialHealthValid, summary.Items[0].Health)
		assert.Equal(t, worker.ComplianceStatusCompliant, summary.ComplianceStatus)
	})

	t.Run("no required types is compliant", func(t *testing.T) {
		summary := worker.BuildCredentialSummary(wrk, []*worker.WorkerCredentialType{twic}, nil, now)
		assert.Empty(t, summary.Items)
		assert.Equal(t, worker.ComplianceStatusCompliant, summary.ComplianceStatus)
	})
}

func TestCredentialProfileField_RoundTrip(t *testing.T) {
	profile := &worker.WorkerProfile{LicenseExpiry: 0}

	assert.Nil(t, worker.CredentialProfileFieldLicenseExpiry.Expiry(profile))
	worker.CredentialProfileFieldLicenseExpiry.Apply(profile, ptr(123), "ABC")
	assert.Equal(t, int64(123), profile.LicenseExpiry)
	assert.Equal(t, "ABC", profile.LicenseNumber)

	worker.CredentialProfileFieldLicenseExpiry.Apply(profile, nil, "")
	assert.Equal(t, int64(123), profile.LicenseExpiry, "clearing the CDL expiry is refused")

	worker.CredentialProfileFieldMedicalCardExpiry.Apply(profile, ptr(456), "")
	assert.Equal(t, int64(456), *worker.CredentialProfileFieldMedicalCardExpiry.Expiry(profile))
	worker.CredentialProfileFieldMedicalCardExpiry.Apply(profile, nil, "")
	assert.Nil(t, profile.MedicalCardExpiry)

	assert.Nil(t, worker.CredentialProfileFieldNone.Expiry(profile))
	assert.True(t, worker.CredentialProfileFieldLicenseExpiry.CarriesNumber())
	assert.False(t, worker.CredentialProfileFieldTWICExpiry.CarriesNumber())
}

func TestWorkerCredential_ValidateAgainstType(t *testing.T) {
	cdl := credType("CDL", true)
	cdl.RequiresNumber = true
	cdl.ProfileField = worker.CredentialProfileFieldLicenseExpiry

	cred := &worker.WorkerCredential{}
	multiErr := errortypes.NewMultiError()
	cred.ValidateAgainstType(cdl, multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Len(t, multiErr.Errors, 2)

	cred.Number = "X1"
	cred.ExpiresAt = ptr(10)
	multiErr = errortypes.NewMultiError()
	cred.ValidateAgainstType(cdl, multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestSystemCredentialTypes(t *testing.T) {
	types := worker.SystemCredentialTypes()
	require.Len(t, types, 12)

	codes := make(map[string]struct{}, len(types))
	fields := make(map[worker.CredentialProfileField]struct{}, len(types))
	for _, typ := range types {
		_, dup := codes[typ.Code]
		assert.False(t, dup, "duplicate code %s", typ.Code)
		codes[typ.Code] = struct{}{}
		if typ.ProfileField.IsSet() {
			_, dupField := fields[typ.ProfileField]
			assert.False(t, dupField, "profile field %s mirrored twice", typ.ProfileField)
			fields[typ.ProfileField] = struct{}{}
		}
		assert.True(t, typ.Category.IsValid())
	}
	assert.Len(t, fields, 6, "every profile expiry column is mirrored by exactly one type")

	// The three obligations 49 CFR 391.51 makes recurring have to be in the
	// catalog, or a driver qualification file has nothing to report them from.
	for _, code := range []string{"ROAD_TEST", "ANNUAL_REVIEW", "VIOLATION_CERT"} {
		_, present := codes[code]
		assert.True(t, present, "%s is required by the driver qualification file", code)
	}
}
