package assistantservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type titlingArtifactRepo struct {
	*assistanttest.StubArtifactRepo

	titles map[pulid.ID]string
	err    error
	asked  [][]pulid.ID
}

func (r *titlingArtifactRepo) ListTitles(
	_ context.Context,
	req *assistantartifactports.ArtifactTitlesRequest,
) (map[pulid.ID]string, error) {
	r.asked = append(r.asked, req.IDs)
	if r.err != nil {
		return nil, r.err
	}
	found := make(map[pulid.ID]string, len(req.IDs))
	for _, id := range req.IDs {
		if title, ok := r.titles[id]; ok {
			found[id] = title
		}
	}

	return found, nil
}

// Every reply names an artifact by its title, whatever words the model wrote:
// the Dispatch desk called one shipment's card "shipment card", "Shipment
// S26100090038417" and nothing at all across three turns of one conversation.
func TestArtifactRecorder_NamesEveryLinkByItsArtifactsTitle(t *testing.T) {
	t.Parallel()

	earlier := pulid.MustNew("art_")
	repo := &titlingArtifactRepo{
		StubArtifactRepo: &assistanttest.StubArtifactRepo{},
		titles:           map[pulid.ID]string{earlier: "Shipments"},
	}
	recorder := keepRecorder(t, repo)
	card, _ := recorder.Observe(assistanttest.Observation("shipment_get", map[string]any{
		"id": "shp_1", "proNumber": "S26100090038417",
	}))
	require.NotNil(t, card)

	reply := "Its [shipment card](artifact:" + card.ID.String() + ") has the details, " +
		"from the [completed loads table](artifact:" + earlier.String() + "); the " +
		"[brief](artifact:art_01MADEUP0000000000000000000) has more."
	messages := []conversation.Message{
		{Role: conversation.RoleUser, Content: "[keep](artifact:" + card.ID.String() + ")"},
		{Role: conversation.RoleAssistant, Content: reply},
	}
	final := reply

	recorder.NameLinks(messages, &final)

	want := "Its [Shipment S26100090038417](artifact:" + card.ID.String() + ") has the " +
		"details, from the [Shipments](artifact:" + earlier.String() + "); the brief has more."
	assert.Equal(t, want, messages[1].Content)
	assert.Equal(t, want, final)
	assert.Equal(t, "[keep](artifact:"+card.ID.String()+")", messages[0].Content,
		"only replies are named; what the person wrote is theirs")
	require.Len(t, repo.asked, 1, "the earlier artifacts are read together, once")
	assert.ElementsMatch(t,
		[]pulid.ID{earlier, pulid.ID("art_01MADEUP0000000000000000000")}, repo.asked[0])
}

// A turn that names only what it made reads nothing.
func TestArtifactRecorder_NamesTheTurnsOwnArtifactsWithoutAQuery(t *testing.T) {
	t.Parallel()

	repo := &titlingArtifactRepo{StubArtifactRepo: &assistanttest.StubArtifactRepo{}}
	recorder := keepRecorder(t, repo)
	card, _ := recorder.Observe(assistanttest.Observation("customer_get", map[string]any{
		"id": "cus_1", "name": "Acme",
	}))
	require.NotNil(t, card)
	messages := []conversation.Message{{
		Role:    conversation.RoleAssistant,
		Content: "See [the card](artifact:" + card.ID.String() + ").",
	}}

	recorder.NameLinks(messages, nil)

	assert.Equal(t, "See [Customer Acme](artifact:"+card.ID.String()+").", messages[0].Content)
	assert.Empty(t, repo.asked)
}

// When the titles cannot be read, a link to an earlier artifact keeps its
// words and still opens; it is never dropped for a failure that is not its own.
func TestArtifactRecorder_KeepsEarlierLinksWhenTheirTitlesCannotBeRead(t *testing.T) {
	t.Parallel()

	earlier := pulid.MustNew("art_")
	repo := &titlingArtifactRepo{
		StubArtifactRepo: &assistanttest.StubArtifactRepo{},
		err:              errors.New("connection reset"),
	}
	recorder := keepRecorder(t, repo)
	content := "See [the loads](artifact:" + earlier.String() + ")."
	messages := []conversation.Message{{Role: conversation.RoleAssistant, Content: content}}

	recorder.NameLinks(messages, nil)

	assert.Equal(t, content, messages[0].Content)
}
