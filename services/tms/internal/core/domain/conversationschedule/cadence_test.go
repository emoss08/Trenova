package conversationschedule

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A message read as a schedule reads the way the Desk composer reads it: the
cadence word, an optional time, and the request after it, with the label the
card shows and a cron Temporal fires on.
*/
func TestParseRequest(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		text    string
		cadence string
		cron    string
		prompt  string
	}{
		{
			name:    "weekday with a time",
			text:    "every weekday at 7:30am, what's blocking the billing queue?",
			cadence: "Every weekday · 7:30 AM",
			cron:    "30 7 * * 1-5",
			prompt:  "What's blocking the billing queue?",
		},
		{
			name:    "slash command as the composer fills it",
			text:    "/schedule every Monday at 8am, summarize last week's detention by customer",
			cadence: "Every Monday · 8:00 AM",
			cron:    "0 8 * * 1",
			prompt:  "Summarize last week's detention by customer",
		},
		{
			name:    "no time runs at eight in the morning",
			text:    "every day check for late loads",
			cadence: "Every day · 8:00 AM",
			cron:    "0 8 * * *",
			prompt:  "Check for late loads",
		},
		{
			name:    "a morning is every day",
			text:    "Each morning at 6:15 am: please list unassigned loads",
			cadence: "Every day · 6:15 AM",
			cron:    "15 6 * * *",
			prompt:  "List unassigned loads",
		},
		{
			name:    "a week runs on Mondays",
			text:    "every week at 9 am, how did we do on margin?",
			cadence: "Every Monday · 9:00 AM",
			cron:    "0 9 * * 1",
			prompt:  "How did we do on margin?",
		},
		{
			name:    "plural day names",
			text:    "every fridays at 4:30 PM wrap up the week",
			cadence: "Every Friday · 4:30 PM",
			cron:    "30 16 * * 5",
			prompt:  "Wrap up the week",
		},
		{
			name:    "the weekend days",
			text:    "every sunday at 12pm, what's due Monday?",
			cadence: "Every Sunday · 12:00 PM",
			cron:    "0 12 * * 0",
			prompt:  "What's due Monday?",
		},
		{
			name:    "midnight",
			text:    "every saturday at 12am, close out the week",
			cadence: "Every Saturday · 12:00 AM",
			cron:    "0 0 * * 6",
			prompt:  "Close out the week",
		},
		{
			name:    "a 24-hour time",
			text:    "every weekday at 17:45 who is still on the road",
			cadence: "Every weekday · 5:45 PM",
			cron:    "45 17 * * 1-5",
			prompt:  "Who is still on the road",
		},
		{
			name:    "a time written without its colon",
			text:    "every tuesday at 730am, aging report",
			cadence: "Every Tuesday · 7:30 AM",
			cron:    "30 7 * * 2",
			prompt:  "Aging report",
		},
		{
			name:    "a colon after the time",
			text:    "every wednesday at 8: what's late?",
			cadence: "Every Wednesday · 8:00 AM",
			cron:    "0 8 * * 3",
			prompt:  "What's late?",
		},
		{
			name:    "am is not read out of the next word",
			text:    "every thursday at 8 amazing loads to review",
			cadence: "Every Thursday · 8:00 AM",
			cron:    "0 8 * * 4",
			prompt:  "Amazing loads to review",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.True(t, IsRequest(tt.text))
			got, err := ParseRequest(tt.text)
			require.NoError(t, err)
			assert.Equal(t, tt.cadence, got.Cadence)
			assert.Equal(t, tt.cron, got.CronExpression)
			assert.Equal(t, tt.prompt, got.Prompt)
			require.NoError(t, cronutils.Validate(got.CronExpression))
		})
	}
}

/*
A question that only happens to start with "every" is a question. Only the
/schedule command insists, and then a cadence it cannot read is an error the
person can fix rather than a turn the agent answers once.
*/
func TestIsRequestLeavesOrdinaryQuestionsAlone(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"Every time I open the queue it is slow, why?",
		"every hour check the board",
		"What happens every Monday?",
		"everyday is fine",
	} {
		assert.False(t, IsRequest(text), text)
	}

	assert.True(t, IsRequest("/schedule"))
	assert.True(t, IsRequest("/schedule hourly, check the board"))
	_, err := ParseRequest("/schedule hourly, check the board")
	require.Error(t, err)
}

func TestParseRequestRefusesWhatItCannotRun(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		text string
	}{
		{name: "no request", text: "every weekday at 7:30am"},
		{name: "no request after the comma", text: "every day, please "},
		{name: "an hour past the clock", text: "every day at 13pm, check loads"},
		{name: "minutes past the hour", text: "every day at 7:75, check loads"},
		{name: "a 24-hour time past midnight", text: "every day at 25:00 check loads"},
		{name: "zero o'clock am", text: "every day at 0am check loads"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseRequest(tt.text)
			require.Error(t, err)
		})
	}
}

/*
The next run is read on the schedule's own clock: 7:30 on a weekday in
Chicago, wherever the server is.
*/
func TestNextRunsOnTheSchedulesClock(t *testing.T) {
	t.Parallel()

	schedule := &Schedule{CronExpression: "30 7 * * 1-5", Timezone: "America/Chicago"}
	// Saturday 3 October 2026, noon in Chicago.
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	saturday := time.Date(2026, time.October, 3, 12, 0, 0, 0, chicago)

	next := schedule.Next(saturday.Unix())
	require.NotNil(t, next)
	assert.Equal(t,
		time.Date(2026, time.October, 5, 7, 30, 0, 0, chicago).Unix(),
		*next,
	)
}
