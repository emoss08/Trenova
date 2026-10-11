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
