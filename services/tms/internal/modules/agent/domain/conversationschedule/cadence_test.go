package conversationschedule

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/pulid"
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
		zone    string
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
		{
			name:    "everyday as one word, on a clock it names",
			text:    "everyday at 9:30 AM EST, give me today's operations board",
			cadence: "Every day · 9:30 AM",
			cron:    "30 9 * * *",
			prompt:  "Give me today's operations board",
			zone:    "America/New_York",
		},
		{
			name:    "daily with a separator",
			text:    "daily: unbilled loads",
			cadence: "Every day · 8:00 AM",
			cron:    "0 8 * * *",
			prompt:  "Unbilled loads",
		},
		{
			name:    "daily under the command",
			text:    "/schedule daily check for late loads",
			cadence: "Every day · 8:00 AM",
			cron:    "0 8 * * *",
			prompt:  "Check for late loads",
		},
		{
			name:    "a region's clock written out",
			text:    "every Monday at 8am Pacific time summarize detention",
			cadence: "Every Monday · 8:00 AM",
			cron:    "0 8 * * 1",
			prompt:  "Summarize detention",
			zone:    "America/Los_Angeles",
		},
		{
			name:    "a 24-hour time in UTC",
			text:    "every weekday at 17:00 UTC: who is still on the road",
			cadence: "Every weekday · 5:00 PM",
			cron:    "0 17 * * 1-5",
			prompt:  "Who is still on the road",
			zone:    "UTC",
		},
		{
			name:    "a clock is read only right after the time",
			text:    "every day at 8, ET loads still open",
			cadence: "Every day · 8:00 AM",
			cron:    "0 8 * * *",
			prompt:  "ET loads still open",
		},
		{
			name:    "a word after the time that names no clock is the request",
			text:    "every day at 8 give me the dispatch board",
			cadence: "Every day · 8:00 AM",
			cron:    "0 8 * * *",
			prompt:  "Give me the dispatch board",
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
			assert.Equal(t, tt.zone, got.Timezone)
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
		"Daily revenue for last week?",
		"daily active drivers by terminal",
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
		{
			name: "two schedules in one message",
			text: "everyday at 10:15am Pacific time, list shipments still not dispatched\n" +
				"daily: unbilled loads older than 3 days",
		},
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

func TestParseRequestInSpanish(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		text    string
		cadence string
		cron    string
		prompt  string
	}{
		{
			name:    "every weekday with a time",
			text:    "cada día laborable a las 7:30, ¿qué está bloqueando la cola de facturación?",
			cadence: "Every weekday · 7:30 AM",
			cron:    "30 7 * * 1-5",
			prompt:  "¿qué está bloqueando la cola de facturación?",
		},
		{
			name:    "business days without accents",
			text:    "todos los dias habiles a las 8 am: cargas sin asignar",
			cadence: "Every weekday · 8:00 AM",
			cron:    "0 8 * * 1-5",
			prompt:  "Cargas sin asignar",
		},
		{
			name:    "the slash command with a hint-shaped time",
			text:    "/schedule todos los lunes a las 8 a. m., resume la detención por cliente",
			cadence: "Every Monday · 8:00 AM",
			cron:    "0 8 * * 1",
			prompt:  "Resume la detención por cliente",
		},
		{
			name:    "no time runs at eight",
			text:    "cada día revisa las cargas atrasadas",
			cadence: "Every day · 8:00 AM",
			cron:    "0 8 * * *",
			prompt:  "Revisa las cargas atrasadas",
		},
		{
			name:    "mornings with please",
			text:    "todas las mañanas a las 6:15, por favor lista las cargas sin asignar",
			cadence: "Every day · 6:15 AM",
			cron:    "15 6 * * *",
			prompt:  "Lista las cargas sin asignar",
		},
		{
			name:    "a week runs on Mondays",
			text:    "cada semana a las 9, ¿cómo nos fue con el margen?",
			cadence: "Every Monday · 9:00 AM",
			cron:    "0 9 * * 1",
			prompt:  "¿cómo nos fue con el margen?",
		},
		{
			name:    "the afternoon and half past",
			text:    "todos los viernes a las 4 y media de la tarde cierra la semana",
			cadence: "Every Friday · 4:30 PM",
			cron:    "30 16 * * 5",
			prompt:  "Cierra la semana",
		},
		{
			name:    "one o'clock and plural Saturdays without the accent",
			text:    "todos los sabados a la 1 pm, ¿qué vence el lunes?",
			cadence: "Every Saturday · 1:00 PM",
			cron:    "0 13 * * 6",
			prompt:  "¿qué vence el lunes?",
		},
		{
			name:    "Wednesday with its accent and midnight at night",
			text:    "cada miércoles a las 12 de la noche: cierre",
			cadence: "Every Wednesday · 12:00 AM",
			cron:    "0 0 * * 3",
			prompt:  "Cierre",
		},
		{
			name:    "a 24-hour time on Sundays",
			text:    "Todos Los Domingos a las 17:45 quién sigue en ruta",
			cadence: "Every Sunday · 5:45 PM",
			cron:    "45 17 * * 0",
			prompt:  "Quién sigue en ruta",
		},
		{
			name:    "a quarter past in the morning",
			text:    "cada martes a las 7 y cuarto de la mañana, informe de antigüedad",
			cadence: "Every Tuesday · 7:15 AM",
			cron:    "15 7 * * 2",
			prompt:  "Informe de antigüedad",
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
		})
	}
}

func TestParseRequestInChinese(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		text    string
		cadence string
		cron    string
		prompt  string
	}{
		{
			name:    "every day at a time",
			text:    "每天8点，汇总延误的运单",
			cadence: "Every day · 8:00 AM",
			cron:    "0 8 * * *",
			prompt:  "汇总延误的运单",
		},
		{
			name:    "a morning with no separator",
			text:    "每天早上7点半汇总未分配的货物",
			cadence: "Every day · 7:30 AM",
			cron:    "30 7 * * *",
			prompt:  "汇总未分配的货物",
		},
		{
			name:    "simplified weekday",
			text:    "每个工作日上午7:30，计费队列卡在哪里？",
			cadence: "Every weekday · 7:30 AM",
			cron:    "30 7 * * 1-5",
			prompt:  "计费队列卡在哪里？",
		},
		{
			name:    "traditional weekday with a full-width colon",
			text:    "每個工作日 17：45：誰還在路上？",
			cadence: "Every weekday · 5:45 PM",
			cron:    "45 17 * * 1-5",
			prompt:  "誰還在路上？",
		},
		{
			name:    "the slash command as the hint is written",
			text:    "/schedule 每周一上午 8 点, 请汇总上周各客户的滞留",
			cadence: "Every Monday · 8:00 AM",
			cron:    "0 8 * * 1",
			prompt:  "汇总上周各客户的滞留",
		},
		{
			name:    "traditional week day in the afternoon",
			text:    "每週五下午4點30分 總結本週",
			cadence: "Every Friday · 4:30 PM",
			cron:    "30 16 * * 5",
			prompt:  "總結本週",
		},
		{
			name:    "a week names Monday and a separator alone marks it",
			text:    "每周：利润怎么样？",
			cadence: "Every Monday · 8:00 AM",
			cron:    "0 8 * * 1",
			prompt:  "利润怎么样？",
		},
		{
			name:    "Sunday spelled with 天 and Chinese numerals",
			text:    "每星期天晚上九点、下周到期的有哪些？",
			cadence: "Every Sunday · 9:00 PM",
			cron:    "0 21 * * 0",
			prompt:  "下周到期的有哪些？",
		},
		{
			name:    "every Saturday with a measure word",
			text:    "每个星期六中午12点，结算本周",
			cadence: "Every Saturday · 12:00 PM",
			cron:    "0 12 * * 6",
			prompt:  "结算本周",
		},
		{
			name:    "the slash command needs no time",
			text:    "/schedule 每日汇总运单",
			cadence: "Every day · 8:00 AM",
			cron:    "0 8 * * *",
			prompt:  "汇总运单",
		},
		{
			name:    "weekdays phrased the other way round with a 24-hour time",
			text:    "工作日每天20点 谁还在路上",
			cadence: "Every weekday · 8:00 PM",
			cron:    "0 20 * * 1-5",
			prompt:  "谁还在路上",
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
		})
	}
}

/*
每天 and cada open ordinary questions too. Spanish needs a unit word like
English does; Chinese also needs the /schedule command, a time or a
separator right after the cadence.
*/
func TestIsRequestLeavesOrdinaryQuestionsAloneInEveryLanguage(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"cada vez que abro la cola va lento, ¿por qué?",
		"cada hora revisa el tablero",
		"¿Qué pasa cada lunes?",
		"cada diario cuenta",
		"todos los clientes, ¿quién debe más?",
		"每天有多少票货？",
		"每天早上有多少票货？",
		"每周一次汇总",
		"每日报告在哪里？",
		"每个司机每天跑多少？",
		"今天每天8点",
	} {
		assert.False(t, IsRequest(text), text)
		_, err := ParseRequest(text)
		require.Error(t, err, text)
	}
}

func TestParseRequestRefusesWhatItCannotRunInEveryLanguage(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"cada día a las 7:30",
		"cada día a las 13 pm, revisa las cargas",
		"cada día a las 25:00 revisa las cargas",
		"cada día a las 8:15 y media revisa",
		"每天8点，",
		"每天25点，汇总",
		"每天下午13点，汇总",
		"/schedule 每天",
	} {
		_, err := ParseRequest(text)
		require.Error(t, err, text)
	}
}

/*
Editing a schedule changes what it asks, how often and when, from the fields
the editor sends. Each cadence the editor offers is one a message can ask for,
so the label and cron match what typing it would have made.
*/
func TestChange_TakesTheRequestAndCadenceAnEditGives(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		edit    Edit
		cadence string
		cron    string
		prompt  string
	}{
		{
			name:    "one day of the week",
			edit:    Edit{Prompt: "  send me delivered shipments ", Repeat: "friday", Time: "10:15"},
			cadence: "Every Friday · 10:15 AM",
			cron:    "15 10 * * 5",
			prompt:  "Send me delivered shipments",
		},
		{
			name:    "every weekday in the afternoon",
			edit:    Edit{Prompt: "Unbilled loads", Repeat: "Weekday", Time: "17:05"},
			cadence: "Every weekday · 5:05 PM",
			cron:    "5 17 * * 1-5",
			prompt:  "Unbilled loads",
		},
		{
			name:    "every day at midnight",
			edit:    Edit{Prompt: "Yesterday's exceptions", Repeat: "day", Time: "00:00"},
			cadence: "Every day · 12:00 AM",
			cron:    "0 0 * * *",
			prompt:  "Yesterday's exceptions",
		},
		{
			name:    "a single-digit hour",
			edit:    Edit{Prompt: "Dock schedule", Repeat: "sunday", Time: "7:30"},
			cadence: "Every Sunday · 7:30 AM",
			cron:    "30 7 * * 0",
			prompt:  "Dock schedule",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schedule := editable()
			multiErr := errortypes.NewMultiError()
			schedule.Change(tt.edit, multiErr)

			require.False(t, multiErr.HasErrors(), multiErr.Error())
			assert.Equal(t, tt.prompt, schedule.Prompt)
			assert.Equal(t, tt.cadence, schedule.Cadence)
			assert.Equal(t, tt.cron, schedule.CronExpression)
			assert.Equal(t, "America/Chicago", schedule.Timezone, "the schedule keeps its clock")
		})
	}
}

/*
An edit that cannot be kept changes nothing, and says which field is wrong.
*/
func TestChange_RefusesWhatItCannotKeep(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		edit  Edit
		field string
	}{
		{
			name:  "no request",
			edit:  Edit{Prompt: "   ", Repeat: "day", Time: "08:00"},
			field: "prompt",
		},
		{
			name:  "a request too long",
			edit:  Edit{Prompt: strings.Repeat("a", MaxPromptLength+1), Repeat: "day", Time: "08:00"},
			field: "prompt",
		},
		{
			name:  "a cadence it does not offer",
			edit:  Edit{Prompt: "x", Repeat: "monthly", Time: "08:00"},
			field: "repeat",
		},
		{
			name:  "an hour past the day",
			edit:  Edit{Prompt: "x", Repeat: "day", Time: "24:00"},
			field: "time",
		},
		{
			name:  "minutes past the hour",
			edit:  Edit{Prompt: "x", Repeat: "day", Time: "08:60"},
			field: "time",
		},
		{
			name:  "a time as words",
			edit:  Edit{Prompt: "x", Repeat: "day", Time: "8am"},
			field: "time",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schedule := editable()
			before := *schedule
			multiErr := errortypes.NewMultiError()
			schedule.Change(tt.edit, multiErr)

			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, tt.field, multiErr.Errors[0].Field)
			assert.Equal(t, before.Prompt, schedule.Prompt)
			assert.Equal(t, before.Cadence, schedule.Cadence)
			assert.Equal(t, before.CronExpression, schedule.CronExpression)
		})
	}
}

func editable() *Schedule {
	return &Schedule{
		ID:             pulid.MustNew("csch_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ThreadID:       pulid.MustNew("athr_"),
		UserID:         pulid.MustNew("usr_"),
		Prompt:         "Send me invoices more than 60 days past due",
		Cadence:        "Every Monday · 8:00 AM",
		CronExpression: "0 8 * * 1",
		Timezone:       "America/Chicago",
		Enabled:        true,
	}
}

/*
A clock is named the way people say it, or as the IANA name an agent sends.
An abbreviation names its region's clock in both halves of the year, so
"EST" in July is still New York's 9:30.
*/
func TestResolveZone(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]string{
		"EST":                   "America/New_York",
		"edt":                   "America/New_York",
		"Eastern time":          "America/New_York",
		"Central Standard Time": "America/Chicago",
		"PT":                    "America/Los_Angeles",
		"gmt":                   "UTC",
		"America/Denver":        "America/Denver",
		"Europe/London":         "Europe/London",
	} {
		got, ok := ResolveZone(name)
		assert.True(t, ok, name)
		assert.Equal(t, want, got, name)
	}

	for _, name := range []string{"", "Local", "Mars/Olympus", "EST and more", "tomorrow"} {
		_, ok := ResolveZone(name)
		assert.False(t, ok, name)
	}
}

/*
A schedule nobody typed is written on its card as the request a person would
have typed for it, in the words the composer reads back as a schedule.
*/
func TestAskedReadsBackAsTheSameSchedule(t *testing.T) {
	t.Parallel()

	schedule := editable()
	multiErr := errortypes.NewMultiError()
	schedule.Change(Edit{Prompt: "give me today's operations board", Repeat: "day", Time: "09:30"}, multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())

	asked := schedule.Asked()
	assert.Equal(t, "Every day at 9:30 AM, Give me today's operations board", asked)

	parsed, err := ParseRequest(asked)
	require.NoError(t, err)
	assert.Equal(t, schedule.Cadence, parsed.Cadence)
	assert.Equal(t, schedule.CronExpression, parsed.CronExpression)
	assert.Equal(t, schedule.Prompt, parsed.Prompt)
}

/*
A request that runs over several lines is one request. Only a later line that
opens with a cadence of its own is a second schedule, which is refused rather
than asked as part of the first.
*/
func TestParseRequestKeepsAManyLineRequestAndRefusesASecondSchedule(t *testing.T) {
	t.Parallel()

	got, err := ParseRequest("every Monday at 8am, summarize:\n- detention by customer\n- late loads")
	require.NoError(t, err)
	assert.Equal(t, "Summarize:\n- detention by customer\n- late loads", got.Prompt)

	_, err = ParseRequest("every day at 8, unbilled loads\nevery weekday at 7:30am CT, what's blocking?")
	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "content", invalid.Field)
	assert.Contains(t, invalid.Error(), "every weekday at 7:30am CT, what's blocking?")
}
