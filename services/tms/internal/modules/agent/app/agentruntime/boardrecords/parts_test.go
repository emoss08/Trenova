package boardrecords_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/boardrecords"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

const (
	jane       = "wrk_01M7V48DVVQGDF0KG409BJH1FJ"
	janeCard   = "wcred_01M7V48E7D6K4KRNHMBA7X53YW"
	janeHealth = "wcred_01M7V48E7E1FF9PPHATT0922WK"
)

func credentialsRead() boardrecords.Read {
	return boardrecords.Of([]string{
		toolresult.Fence("worker_list_credentials", `{"count":2,"items":[`+
			`{"id":"`+janeCard+`","credentialType":{"id":"wct_01M7V48E5HERT6ZQN8GJBGY46C",`+
			`"name":"DOT Medical Card"},"worker":{"id":"`+jane+`","wholeName":"Jane Doe"}},`+
			`{"id":"`+janeHealth+`","worker":{"id":"`+jane+`","wholeName":"Jane Doe"}}]}`),
	})
}

func workers(ids ...string) map[recordkind.Kind][]pulid.ID {
	out := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		out = append(out, pulid.ID(id))
	}

	return map[recordkind.Kind][]pulid.ID{recordkind.Kind(permission.ResourceWorker): out}
}

func TestPartsRefusal_SendsABoardOfWorkersAboutCredentialsToTheCredentials(t *testing.T) {
	t.Parallel()

	refusal := credentialsRead().PartsRefusal("show_answer", workers(jane),
		"Jane Doe's credentials have lapsed.")

	assert.Contains(t, refusal, "its words are about worker credentials, but its records are "+
		"workers")
	assert.Contains(t, refusal, janeCard+", "+janeHealth)
}

func TestPartsRefusal_LeavesWorkersAloneWhenTheWordsAreAboutThem(t *testing.T) {
	t.Parallel()

	read := credentialsRead()

	assert.Empty(t, read.PartsRefusal("show_answer", workers(jane),
		"Jane Doe is off dispatch today."))
	assert.Empty(t, read.PartsRefusal("show_answer", workers("wrk_01M7V48DWSAX7AFMQ3W2SHBHHR"),
		"Robert Brown's credentials are current."), "no credential of his was read")
}

func TestWholeTyped_TakesACountOfTheShownWhenTheWordsTypeTheListsTotal(t *testing.T) {
	t.Parallel()

	read := boardrecords.Of([]string{
		toolresult.Fence("shipment_list", `{"count":2,"total":1290,"hasMore":true,"items":[`+
			`{"id":"`+shp008+`","proNumber":"SEED-SHP-008"},`+
			`{"id":"`+shp010+`","proNumber":"SEED-SHP-010"}]}`),
	})

	assert.True(t, read.WholeTyped("There are 1,290 late loads; {count} are live.", 2))
	assert.True(t, read.WholeTyped("There are 1290 late loads; {count} are live.", 2))
	assert.False(t, read.WholeTyped("There are {count} late loads.", 2))
}

func TestReplaceIDs_SwapsEveryIDInTheRecordsAndTheFigures(t *testing.T) {
	t.Parallel()

	replaced := boardrecords.ReplaceIDs(map[string]any{
		"answer": "It brings in {revenue:" + shp001 + "}.",
		"groups": []any{map[string]any{"records": []any{shp001, shp008}}},
	}, map[string]string{shp001: shp010})

	assert.Equal(t, "It brings in {revenue:"+shp010+"}.", replaced["answer"])
	assert.Equal(t, []any{shp010, shp008},
		replaced["groups"].([]any)[0].(map[string]any)["records"])
}

func TestWholesOf_PassesWhoseEachShownPartIs(t *testing.T) {
	t.Parallel()

	arguments := map[string]any{
		"answer": "{count} credentials across three drivers have lapsed.",
		"groups": []any{map[string]any{"title": "Lapsed", "records": []any{janeCard, janeHealth}}},
	}
	withWholes, parted := credentialsRead().WholesOf(arguments)

	assert.True(t, parted, "final-a credentials #1 typed a count of people beside credentials")
	assert.Equal(t, map[string]any{janeCard: jane, janeHealth: jane}, withWholes["partOf"])
	assert.NotContains(t, arguments, "partOf", "the call is copied, not changed in place")

	shipments := map[string]any{
		"answer": "{count} loads.",
		"groups": []any{map[string]any{"records": []any{"shp_01M7V48EQGE9TBM60WF4JKM3VQ"}}},
	}
	_, parted = credentialsRead().WholesOf(shipments)
	assert.False(t, parted, "a shipment is part of nothing")
}
