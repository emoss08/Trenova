package agentquality

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func automaticCase(id, tool string, createdAt int64) AutomaticCase {
	return AutomaticCase{ID: pulid.ID(id), Tool: tool, CreatedAt: createdAt}
}

func TestAutomaticOverflow_KeepsEverythingWithinTheLimit(t *testing.T) {
	t.Parallel()

	cases := []AutomaticCase{
		automaticCase("aec_1", "update_rate", 10),
		automaticCase("aec_2", "update_rate", 20),
	}

	assert.Empty(t, AutomaticOverflow(cases, 2))
	assert.Empty(t, AutomaticOverflow(cases, 5))
	assert.Empty(t, AutomaticOverflow(nil, 3))
}

func TestAutomaticOverflow_RetiresTheOldestOfOneTool(t *testing.T) {
	t.Parallel()

	cases := []AutomaticCase{
		automaticCase("aec_1", "update_rate", 10),
		automaticCase("aec_2", "update_rate", 30),
		automaticCase("aec_3", "update_rate", 20),
	}

	assert.Equal(t, []pulid.ID{"aec_1"}, AutomaticOverflow(cases, 2))
}

func TestAutomaticOverflow_ABusyToolDoesNotCrowdOutAQuietOne(t *testing.T) {
	t.Parallel()

	cases := []AutomaticCase{
		automaticCase("aec_hold_old", "shipment_place_hold", 1),
		automaticCase("aec_rate_1", "update_rate", 10),
		automaticCase("aec_rate_2", "update_rate", 20),
		automaticCase("aec_rate_3", "update_rate", 30),
		automaticCase("aec_rate_4", "update_rate", 40),
	}

	overflow := AutomaticOverflow(cases, 3)

	assert.ElementsMatch(t, []pulid.ID{"aec_rate_1", "aec_rate_2"}, overflow,
		"the quiet tool's only case outlives the busy tool's older ones")
}

func TestAutomaticOverflow_SharesTheLimitAcrossToolsNewestFirst(t *testing.T) {
	t.Parallel()

	cases := []AutomaticCase{
		automaticCase("aec_a1", "a", 1),
		automaticCase("aec_a2", "a", 2),
		automaticCase("aec_a3", "a", 3),
		automaticCase("aec_b1", "b", 4),
		automaticCase("aec_b2", "b", 5),
		automaticCase("aec_b3", "b", 6),
	}

	assert.ElementsMatch(t,
		[]pulid.ID{"aec_a1", "aec_a2", "aec_b1", "aec_b2"},
		AutomaticOverflow(cases, 2),
		"each tool keeps its newest case",
	)
	assert.ElementsMatch(t,
		[]pulid.ID{"aec_a1", "aec_a2", "aec_b1"},
		AutomaticOverflow(cases, 3),
		"the odd place goes to the newer of the two second-newest cases",
	)
}

func TestAutomaticOverflow_TiesBreakOnIDSoTheSameCasesGoEveryTime(t *testing.T) {
	t.Parallel()

	cases := []AutomaticCase{
		automaticCase("aec_1", "update_rate", 10),
		automaticCase("aec_2", "update_rate", 10),
	}
	reversed := []AutomaticCase{cases[1], cases[0]}

	assert.Equal(t, []pulid.ID{"aec_1"}, AutomaticOverflow(cases, 1))
	assert.Equal(t, []pulid.ID{"aec_1"}, AutomaticOverflow(reversed, 1))
}

func TestAutomaticOverflow_KeepingNoneRetiresAll(t *testing.T) {
	t.Parallel()

	cases := []AutomaticCase{
		automaticCase("aec_1", "a", 1),
		automaticCase("aec_2", "b", 2),
	}

	assert.ElementsMatch(t, []pulid.ID{"aec_1", "aec_2"}, AutomaticOverflow(cases, 0))
	assert.ElementsMatch(t, []pulid.ID{"aec_1", "aec_2"}, AutomaticOverflow(cases, -4))
}
