package agentreflectionservice

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const correctionNow = int64(1_800_000_000)

func corrected(
	agentID pulid.ID,
	record string,
	daysAgo int64,
	fields ...string,
) agentreflectionports.CorrectedWork {
	work := agentreflectionports.CorrectedWork{
		AgentDefinitionID: agentID,
		Resource:          "shipment",
		ResourceID:        record,
		Label:             record,
		ChangedAt:         correctionNow - daysAgo*secondsPerDay,
	}
	for _, field := range fields {
		work.Fields = append(work.Fields, agentreflectionports.CorrectedField{
			Field: field, Was: "14:00", Now: "08:00",
		})
	}

	return work
}

// Three records changed the same way is how the organization does it; one
// record changed three times is someone making up their mind.
func TestCorrectionPatterns_AHabitIsTheSameFieldChangedOnSeveralRecords(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	work := []agentreflectionports.CorrectedWork{
		corrected(agentID, "shp_1", 1, "deliveryWindowStart", "notes"),
		corrected(agentID, "shp_2", 4, "deliveryWindowStart"),
		corrected(agentID, "shp_3", 9, "deliveryWindowStart"),
		corrected(agentID, "shp_4", 1, "serviceType"),
		corrected(agentID, "shp_4", 2, "serviceType"),
		corrected(agentID, "shp_4", 3, "serviceType"),
	}

	habits := CorrectionPatterns(work, correctionNow)

	require.Len(t, habits, 1)
	habit := habits[0]
	assert.Equal(t, "deliveryWindowStart", habit.Field)
	assert.Equal(t, "shipment · delivery window start", habit.What())
	assert.Equal(t, agentID.String()+"|shipment|deliveryWindowStart", habit.Key())
	require.Len(t, habit.Work, 3)
	assert.Equal(t, "shp_1", habit.Work[0].ResourceID, "the newest change comes first")
	assert.Len(t, habit.Work[0].Fields, 1, "each record carries only the field the habit is about")
}

func TestCorrectionPatterns_AHabitThatStoppedOrBelongsToAnotherAgentIsLeftAlone(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("agdef_"), pulid.MustNew("agdef_")

	stale := CorrectionPatterns([]agentreflectionports.CorrectedWork{
		corrected(first, "shp_1", 10, "deliveryWindowStart"),
		corrected(first, "shp_2", 12, "deliveryWindowStart"),
		corrected(first, "shp_3", 14, "deliveryWindowStart"),
	}, correctionNow)
	assert.Empty(t, stale, "nobody has changed it lately")

	split := CorrectionPatterns([]agentreflectionports.CorrectedWork{
		corrected(first, "shp_1", 1, "deliveryWindowStart"),
		corrected(first, "shp_2", 1, "deliveryWindowStart"),
		corrected(second, "shp_3", 1, "deliveryWindowStart"),
	}, correctionNow)
	assert.Empty(
		t,
		split,
		"two agents corrected twice and once are not one agent corrected three times",
	)
}

func TestCorrectionPatterns_KeepsTheNewestChangeOfEachRecordAndTheWidestHabitFirst(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	work := make([]agentreflectionports.CorrectedWork, 0, 40)
	for idx := range maxCorrectionInstances + 5 {
		work = append(
			work,
			corrected(agentID, fmt.Sprintf("shp_%02d", idx), 1, "deliveryWindowStart"),
		)
	}
	for idx := range MinCorrectedRecords {
		work = append(work, corrected(agentID, fmt.Sprintf("shp_%02d", idx), 1, "serviceType"))
	}
	work = append(work, corrected(agentID, "shp_00", 0, "deliveryWindowStart"))

	habits := CorrectionPatterns(work, correctionNow)

	require.Len(t, habits, 2)
	assert.Equal(t, "deliveryWindowStart", habits[0].Field)
	assert.Len(t, habits[0].Work, maxCorrectionInstances)
	assert.Equal(t, "shp_00", habits[0].Work[0].ResourceID)
	assert.Equal(
		t,
		correctionNow,
		habits[0].Work[0].ChangedAt,
		"a record changed twice is read as its latest change",
	)
	assert.Equal(t, "serviceType", habits[1].Field)
}

func TestReadSignals_WhatPeopleKeptChangingIsASignalFromPeople(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	signals := ReadSignals(&Window{
		Corrections: []agentreflectionports.CorrectedWork{
			corrected(agentID, "shp_1", 1, "deliveryWindowStart"),
			corrected(agentID, "shp_2", 1, "deliveryWindowStart"),
			corrected(agentID, "shp_3", 1, "deliveryWindowStart"),
		},
		CorrectedWhat: "shipment · delivery window start",
	})

	require.Len(t, signals, 1)
	assert.Equal(t, agentreflection.ReflectionSignalChangedAfterwards, signals[0].Kind)
	assert.Equal(t, 3, signals[0].Count)
	assert.Equal(t, "shipment · delivery window start", signals[0].Detail)
	assert.True(t, signals.FromPeople(), "a person's change is a person's decision")
	assert.False(t, signals.OnlyInPassing())
}

func TestCorrectionPrompt_ListsWhatStoodAndWhatAPersonMadeIt(t *testing.T) {
	t.Parallel()

	in := &promptInput{
		AgentName: "Dispatch desk",
		Corrected: "shipment · delivery window start",
		Corrections: []agentreflectionports.CorrectedWork{
			corrected(pulid.MustNew("agdef_"), "S-1042", 1, "deliveryWindowStart"),
		},
	}

	system := systemPrompt(in)
	assert.Contains(t, system, "changed by hand")
	assert.Contains(t, system, "Never write a record's id")

	sections := contextSections(in).Sections
	titles := make([]string, 0, len(sections))
	for _, section := range sections {
		titles = append(titles, section.Title)
	}
	assert.Contains(t, titles, "What people changed afterwards: shipment · delivery window start")
	assert.NotContains(t, titles, "The work", "there is no transcript to read")
	assert.Contains(
		t,
		correctionLines(in.Corrections),
		`- S-1042: delivery window start was "14:00" after the agent's work; a person changed it to "08:00"`,
	)
}

func TestLearningRows_JoinWhatLookBacksCostWithWhatBecameOfTheirLessons(t *testing.T) {
	t.Parallel()

	rows := LearningRows(
		[]agentreflectionports.LearningTrigger{
			{
				Trigger:       "PersonTold",
				LookBacks:     40,
				TaughtNothing: 31,
				InputTokens:   90_000,
				OutputTokens:  4_000,
			},
			{
				Trigger:      "PersonCorrected",
				LookBacks:    6,
				ToldAgain:    2,
				InputTokens:  20_000,
				OutputTokens: 900,
			},
			{Trigger: "SomethingRetired", LookBacks: 1},
		},
		[]agentmemoryports.MemoryOutcome{
			{
				Source:  agentmemory.MemorySourceReflection,
				Trigger: "PersonTold",
				Status:  agentmemory.MemoryStatusActive,
				Count:   5,
				Used:    3,
			},
			{
				Source:  agentmemory.MemorySourceReflection,
				Trigger: "PersonTold",
				Status:  agentmemory.MemoryStatusSuggested,
				Count:   2,
			},
			{
				Source:  agentmemory.MemorySourceReflection,
				Trigger: "PersonTold",
				Status:  agentmemory.MemoryStatusDismissed,
				Count:   4,
				Lapsed:  1,
			},
			{
				Source:  agentmemory.MemorySourceAgent,
				Trigger: "",
				Status:  agentmemory.MemoryStatusActive,
				Count:   7,
				Used:    6,
			},
			{
				Source:  agentmemory.MemorySourceAgent,
				Trigger: "",
				Status:  agentmemory.MemoryStatusRetired,
				Count:   1,
			},
			{
				Source:  agentmemory.MemorySourceUser,
				Trigger: "PersonTold",
				Status:  agentmemory.MemoryStatusPaused,
				Count:   1,
			},
		},
	)

	require.Len(t, rows, 5)

	corrected := rows[0]
	require.NotNil(t, corrected.Trigger)
	assert.Equal(t, agentreflection.ReflectionSignalPersonCorrected, *corrected.Trigger)
	assert.Equal(t, 6, corrected.LookBacks)
	assert.Equal(t, 2, corrected.ToldAgain)

	told := rows[1]
	assert.Equal(t, agentreflection.ReflectionSignalPersonTold, *told.Trigger)
	assert.Equal(t, 40, told.LookBacks)
	assert.Equal(t, 31, told.TaughtNothing)
	assert.Equal(t, 5, told.Kept)
	assert.Equal(t, 2, told.Waiting)
	assert.Equal(t, 3, told.Dismissed, "an offer nobody decided lapsed; it was not turned down")
	assert.Equal(t, 1, told.Lapsed)
	assert.Equal(t, 3, told.Used)
	assert.Equal(t, 94_000, told.InputTokens+told.OutputTokens)

	unknown := rows[2]
	assert.Equal(t, agentmemory.MemorySourceReflection, unknown.Source)
	assert.Nil(t, unknown.Trigger, "a trigger no longer known is counted, under no name")

	saved := rows[3]
	assert.Equal(t, agentmemory.MemorySourceAgent, saved.Source)
	assert.Nil(t, saved.Trigger)
	assert.Equal(t, 7, saved.Kept)
	assert.Equal(t, 1, saved.Retired)

	written := rows[4]
	assert.Equal(t, agentmemory.MemorySourceUser, written.Source)
	assert.Nil(t, written.Trigger, "only a look back has a trigger")
	assert.Equal(t, 1, written.Kept)
}
