package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	businessUnitID = "bu_01M37R101VKZTB7TSKR30FJ0AT"
)

func TestClassifyValues_ReadsEachColumnByItsNameAndItsValues(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		key    string
		values []any
		want   assistantartifact.DisplayType
		shown  bool
	}{
		{"a flag that never holds is noise", "stale", []any{false, false}, "", false},
		{"a flag that holds is said", "stale", []any{false, true}, assistantartifact.DisplayFlag, true},
		{"a yes or no", "hazardous", []any{true, false}, assistantartifact.DisplayBoolean, true},
		{"a count named like a date stays a count", "daysUntilExpiry", []any{float64(21)}, assistantartifact.DisplayNumber, true},
		{"an unset date is nothing", "medicalCardExpiry", []any{float64(0)}, "", false},
		{"the phrase for an unset date is words", "medicalCardExpiry", []any{assistanttest.MedicalCardNote}, assistantartifact.DisplayText, true},
		{"a phrase beside real dates is a date column", "medicalCardExpiry", []any{assistanttest.MedicalCardNote, assistanttest.DetectedOn}, assistantartifact.DisplayDate, true},
		{"a date the runtime already wrote", "windowStart", []any{"2026-08-24 11:20 PDT (30 days ago)"}, assistantartifact.DisplayDateTime, true},
		{"an amount due is not a date", "amountDue", []any{"12.00"}, assistantartifact.DisplayText, true},
		{"reason is not a date", "reason", []any{"Driver out sick"}, assistantartifact.DisplayText, true},
		{"a total written as a decimal is money", "total", []any{"12.00"}, assistantartifact.DisplayMoney, true},
		{"a total counted is a number", "total", []any{float64(4)}, assistantartifact.DisplayNumber, true},
		{"a percentage", "onTimePercent", []any{"92.5"}, assistantartifact.DisplayPercent, true},
		{"a year is not a quantity", "year", []any{float64(2019)}, assistantartifact.DisplayText, true},
		{"a state is two letters, not a status", "state", []any{"CA"}, assistantartifact.DisplayText, true},
		{"a status with spaces is words", "assignmentBlocked", []any{"Medical card lapsed"}, assistantartifact.DisplayText, true},
		{"a type is a category", "driverType", []any{"OTR", "Local"}, assistantartifact.DisplayEnum, true},
		{"prose by its name", "recommendation", []any{"Call them."}, assistantartifact.DisplayLongText, true},
		{"words in a list", "tags", []any{[]any{"hazmat", "team"}}, assistantartifact.DisplayText, true},
		{"records nested in a row", "stops", []any{[]any{map[string]any{"city": "Reno"}}}, "", false},
		{"a nested record by its name", "customer", []any{map[string]any{"id": assistanttest.CustomerPulid, "name": "Acme"}}, assistantartifact.DisplayText, true},
		{"a nested record with no name", "rating", []any{map[string]any{"score": float64(3)}}, "", false},
		{"the tenant", "organizationId", []any{assistanttest.OrganizationID}, "", false},
		{"a version", "version", []any{float64(3)}, "", false},
		{"a snake-case id", "business_unit_id", []any{businessUnitID}, "", false},
		{"a word ending in id letters is not an id", "paid", []any{true}, assistantartifact.DisplayBoolean, true},
		{"which way is worse", "direction", []any{"HigherIsWorse"}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, shown := assistantartifact.ClassifyValues(tc.key, tc.values, true)
			assert.Equal(t, tc.shown, shown)
			if tc.shown {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// A bare amount beside a method may be a percentage of the linehaul, so it
// is a figure rather than money.
func TestClassifyValues_AnAmountBesideAMethodIsNotMoney(t *testing.T) {
	t.Parallel()

	got, shown := assistantartifact.ClassifyValues("amount", []any{"12.50"}, false)
	require.True(t, shown)
	assert.NotEqual(t, assistantartifact.DisplayMoney, got)

	got, _ = assistantartifact.ClassifyValues("amount", []any{"12.50"}, true)
	assert.Equal(t, assistantartifact.DisplayMoney, got)
}

func TestDisplayLabel_SaysWhatHappenedRatherThanWhen(t *testing.T) {
	t.Parallel()

	for key, want := range map[string]string{
		"detectedOn":   "Detected",
		"createdAt":    "Created",
		"windowStart":  "Window start",
		"lastMvrCheck": "Last MVR check",
		"cdlClass":     "CDL class",
		"proNumber":    "Pro number",
	} {
		_, displayType := assistantartifact.DatedKey(key)
		if displayType == "" {
			displayType = assistantartifact.DisplayText
		}
		assert.Equal(t, want, assistantartifact.DisplayLabel(key, displayType), key)
	}
}
