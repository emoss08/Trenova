package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
)

func reminderErrors(days []int32, enabled bool) []string {
	bc := &BillingControl{PaymentRemindersEnabled: enabled, PaymentReminderDays: days}
	multiErr := errortypes.NewMultiError()
	bc.validatePaymentReminderDays(multiErr)
	messages := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		messages = append(messages, fieldErr.Message)
	}
	return messages
}

func TestPaymentReminderDays(t *testing.T) {
	t.Parallel()

	assert.Empty(t, reminderErrors([]int32{1, 15, 30}, true))
	assert.Empty(t, reminderErrors(nil, false), "nothing is checked while reminders are off")
	assert.Equal(t, []string{"Choose at least one day past due to send a reminder on"},
		reminderErrors(nil, true))
	assert.Equal(t, []string{"List reminder days from earliest to latest, each once"},
		reminderErrors([]int32{15, 15}, true))
	assert.Equal(t, []string{"List reminder days from earliest to latest, each once"},
		reminderErrors([]int32{30, 1}, true))
	assert.Equal(t, []string{"Reminder days must be between 1 and 365 days past due"},
		reminderErrors([]int32{0, 10}, true))
	assert.Equal(t, []string{"Send at most six reminders per invoice"},
		reminderErrors([]int32{1, 2, 3, 4, 5, 6, 7}, true))
}

func TestBillingControlStartsWithTheDefaultReminderSchedule(t *testing.T) {
	t.Parallel()

	bc := &BillingControl{}
	assert.NoError(t, bc.BeforeAppendModel(t.Context(), &bun.InsertQuery{}))
	assert.Equal(t, []int32{1, 15, 30}, bc.PaymentReminderDays)

	bc.PaymentReminderDays = nil
	assert.NoError(t, bc.BeforeAppendModel(t.Context(), &bun.UpdateQuery{}))
	assert.Equal(t, []int32{1, 15, 30}, bc.PaymentReminderDays,
		"a save that leaves the schedule out keeps a usable one")
}

func TestMonthlyStatementDayMustFallInEveryMonth(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		day   int16
		valid bool
	}{
		{day: 0, valid: true},
		{day: 1, valid: true},
		{day: 28, valid: true},
		{day: 29, valid: false},
		{day: -1, valid: false},
	} {
		bc := &BillingControl{MonthlyStatementsEnabled: true, MonthlyStatementDay: tc.day}
		multiErr := errortypes.NewMultiError()
		bc.Validate(multiErr)
		assert.Equal(t, !tc.valid, hasFieldError(multiErr, "monthlyStatementDay"), "day %d", tc.day)
	}
}

func hasFieldError(multiErr *errortypes.MultiError, field string) bool {
	for _, fieldErr := range multiErr.Errors {
		if fieldErr.Field == field {
			return true
		}
	}
	return false
}
